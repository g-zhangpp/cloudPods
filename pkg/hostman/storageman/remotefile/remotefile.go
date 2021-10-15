// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remotefile

import (
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SImageDesc struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Id     string `json:"id"`
	Chksum string `json:"chksum"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
}

type SRemoteFile struct {
	ctx          context.Context
	url          string
	downloadUrl  string
	localPath    string
	tmpPath      string
	preChksum    string
	compress     bool
	timeout      time.Duration
	extraHeaders map[string]string

	chksum string
	format string
	name   string
}

func NewRemoteFile(
	ctx context.Context, url, localPath string, compress bool,
	PreChksum string, timeout int, extraHeaders map[string]string,
	tmpPath string, downloadUrl string,
) *SRemoteFile {
	if timeout <= 0 {
		timeout = 24 * 3600 //24 hours
	}
	if len(tmpPath) == 0 {
		tmpPath = localPath
	}

	return &SRemoteFile{
		ctx:          ctx,
		url:          url,
		localPath:    localPath,
		compress:     compress,
		preChksum:    PreChksum,
		timeout:      time.Duration(timeout) * time.Second,
		extraHeaders: extraHeaders,
		tmpPath:      tmpPath,
		downloadUrl:  downloadUrl,
	}
}

func (r *SRemoteFile) Fetch() error {
	if len(r.preChksum) > 0 {
		log.Infof("Fetch remote file with precheck sum: %s", r.preChksum)
		return r.fetch(r.preChksum)
	}
	if fileutils2.Exists(r.localPath) {
		err := r.VerifyIntegrity()
		if err != nil {
			log.Warningf("Local path %s file mistmatch, refetch", r.localPath)
			return r.fetch("")
		}
		return nil
	}
	log.Infof("Fetch remote file %s to %s", r.downloadUrl, r.tmpPath)
	return r.fetch("")
}

func (r *SRemoteFile) GetInfo() (*SImageDesc, error) {
	fi, err := os.Stat(r.localPath)
	if err != nil {
		return nil, errors.Wrapf(err, "os.Stat(%s)", r.localPath)
	}

	return &SImageDesc{
		Name:   r.name,
		Format: r.format,
		Chksum: r.chksum,
		Path:   r.localPath,
		Size:   fi.Size(),
	}, nil
}

func (r *SRemoteFile) VerifyIntegrity() error {
	localChksum, err := fileutils2.MD5(r.localPath)
	if err != nil {
		return errors.Wrapf(err, "fileutils2.MD5(%s)", r.localPath)
	}
	if r.preChksum != "" {
		if localChksum == r.preChksum {
			return nil
		}
	}
	err = r.download(false, "")
	if err == nil && localChksum == r.chksum {
		return nil
	}
	log.Warningf("Integrity mistmatch, fetch from remote")
	return r.fetch("")
}

func (r *SRemoteFile) fetch(preChksum string) error {
	var err error
	for i := 0; i < 3; i++ {
		err = r.download(true, preChksum)
		if err == nil {
			if len(r.chksum) > 0 && fileutils2.Exists(r.tmpPath) {
				localChksum, err := fileutils2.MD5(r.tmpPath)
				if err != nil {
					return errors.Wrapf(err, "TmpPath fileutils2.MD5(%s)", r.tmpPath)
				}
				if r.chksum != localChksum {
					return fmt.Errorf("remote checksum %s != local checksum %s", r.chksum, localChksum)
				}
				syscall.Unlink(r.localPath)
				return syscall.Rename(r.tmpPath, r.localPath)
			}
			break
		}
	}
	return errors.Wrapf(err, "download")
}

// retry download
func (r *SRemoteFile) download(getData bool, preChksum string) error {
	if getData {
		// fetch image headers and set resource properties
		err := r.downloadInternal(false, preChksum)
		if err != nil {
			log.Errorf("fetch image properties failed %v", err)
		}
	}
	err := r.downloadInternal(getData, preChksum)
	if err == nil {
		return nil
	}
	log.Errorf("download from cached url %s failed, try direct download from %s ...", r.downloadUrl, r.url)
	r.downloadUrl = ""
	if getData {
		// fetch image headers and set resource properties
		err = r.downloadInternal(false, preChksum)
		if err != nil {
			log.Errorf("fetch image properties failed error: %v", err)
		}
	}
	return r.downloadInternal(getData, preChksum)
}

func (r *SRemoteFile) downloadInternal(getData bool, preChksum string) error {
	var header = http.Header{}
	header.Set("X-Auth-Token", auth.GetTokenString())
	if len(preChksum) > 0 {
		header.Set("X-Image-Meta-Checksum", preChksum)
	}
	if r.compress {
		header.Set("X-Compress-Content", "zlib")
	}
	if len(r.extraHeaders) > 0 {
		for k, v := range r.extraHeaders {
			header.Set(k, v)
		}
	}
	var method, url = "HEAD", r.url
	if len(r.downloadUrl) > 0 {
		url = r.downloadUrl
	}
	if getData {
		method = "GET"
	}

	httpCli := httputils.GetTimeoutClient(r.timeout)
	resp, err := httputils.Request(httpCli, r.ctx,
		httputils.THttpMethod(method), url, header, nil, false)
	if err != nil {
		return errors.Wrapf(err, "request %s %s", method, url)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 300 {
		if getData {
			os.Remove(r.tmpPath)
			fi, err := os.Create(r.tmpPath)
			if err != nil {
				return errors.Wrapf(err, "os.Create(%s)", r.tmpPath)
			}
			defer fi.Close()

			var reader = resp.Body

			if r.compress {
				zlibRC, err := zlib.NewReader(resp.Body)
				if err != nil {
					return errors.Wrapf(err, "zlib.NewReader")
				}
				defer zlibRC.Close()
				reader = zlibRC
			}
			var finishChan = make(chan struct{})
			go func() {
				defer recover()
				for {
					select {
					case <-time.After(10 * time.Second):
						info, err := fi.Stat()
						if err != nil {
							log.Errorf("failed stat file %s", r.tmpPath)
							return
						}
						log.Infof("written file %s size %dM", r.tmpPath, info.Size()/1024/1024)
					case <-finishChan:
						return
					}
				}
			}()
			_, err = io.Copy(fi, reader)
			close(finishChan)
			if err != nil {
				return errors.Wrapf(err, "io.Copy to tmpPath %s from reader", r.tmpPath)
			}
		}
		r.setProperties(resp.Header)
		return nil
	} else if resp.StatusCode == 304 {
		if fileutils2.Exists(r.tmpPath) {
			if err := os.Remove(r.tmpPath); err != nil {
				log.Errorf("Fail to remove file %s", r.tmpPath)
			}
		}
		return nil
	}
	return fmt.Errorf("Remote file fetch %s %s error %d", method, url, resp.StatusCode)
}

func (r *SRemoteFile) setProperties(header http.Header) {
	if chksum := header.Get("X-Image-Meta-Checksum"); len(chksum) > 0 {
		r.chksum = chksum
	}
	if format := header.Get("X-Image-Meta-Disk_format"); len(format) > 0 {
		r.format = format
	}
	if name := header.Get("X-Image-Meta-Name"); len(name) > 0 {
		r.name = name
	}
}
