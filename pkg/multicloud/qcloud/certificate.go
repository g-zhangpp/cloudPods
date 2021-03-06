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

package qcloud

import (
	"crypto/sha1"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type projectInfo struct {
	ProjectID  string `json:"projectId"`
	OwnerUin   int64  `json:"ownerUin"`
	Name       string `json:"name"`
	CreatorUin int64  `json:"creatorUin"`
	CreateTime string `json:"createTime"`
	Info       string `json:"info"`
}

// https://cloud.tencent.com/document/api/400/13675
type SCertificate struct {
	multicloud.SResourceBase
	multicloud.QcloudTags
	region *SRegion

	CertificateID       string      `json:"CertificateId"`
	CertificateType     string      `json:"CertificateType"`
	Deployable          bool        `json:"Deployable"`
	RenewAble           bool        `json:"RenewAble"`
	OwnerUin            string      `json:"ownerUin"`
	ProjectID           string      `json:"projectId"`
	From                string      `json:"from"`
	ProductZhName       string      `json:"productZhName"`
	Domain              string      `json:"domain"`
	Alias               string      `json:"alias"`
	Status              int         `json:"status"`
	VulnerabilityStatus string      `json:"vulnerability_status"`
	CERTBeginTime       time.Time   `json:"certBeginTime"`
	CERTEndTime         time.Time   `json:"certEndTime"`
	ValidityPeriod      string      `json:"validityPeriod"`
	InsertTime          string      `json:"insertTime"`
	ProjectInfo         projectInfo `json:"projectInfo"`
	StatusName          string      `json:"status_name"`
	IsVip               bool        `json:"is_vip"`
	IsDv                bool        `json:"is_dv"`
	IsWildcard          bool        `json:"is_wildcard"`
	IsVulnerability     bool        `json:"is_vulnerability"`

	// certificate details
	detailsInitd          bool     `json:"details_initd"`
	SubjectAltName        []string `json:"subjectAltName"`
	CertificatePrivateKey string   `json:"CertificatePrivateKey"`
	CertificatePublicKey  string   `json:"CertificatePublicKey"`
}

func (self *SCertificate) GetDetails() *SCertificate {
	if !self.detailsInitd {
		cert, err := self.region.GetCertificate(self.GetId())
		if err != nil {
			log.Debugf("GetCertificate %s", err)
		}

		self.detailsInitd = true
		self.SubjectAltName = cert.SubjectAltName
		self.CertificatePrivateKey = cert.CertificatePrivateKey
		self.CertificatePublicKey = cert.CertificatePublicKey
	}

	return self
}

func (self *SCertificate) GetPublickKey() string {
	return self.GetDetails().CertificatePublicKey
}

func (self *SCertificate) GetPrivateKey() string {
	return self.GetDetails().CertificatePrivateKey
}

// ??????????????????
func (self *SCertificate) Sync(name, privateKey, publickKey string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SCertificate) Delete() error {
	return self.region.DeleteCertificate(self.GetId())
}

func (self *SCertificate) GetId() string {
	return self.CertificateID
}

func (self *SCertificate) GetName() string {
	return self.Alias
}

func (self *SCertificate) GetGlobalId() string {
	return self.CertificateID
}

// todo: ????????????onecloud??????????????????
func (self *SCertificate) GetStatus() string {
	return strconv.Itoa(self.Status)
}

func (self *SCertificate) Refresh() error {
	cert, err := self.region.GetCertificate(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetCertificate")
	}

	return jsonutils.Update(self, cert)
}

func (self *SCertificate) IsEmulated() bool {
	return false
}

func (self *SCertificate) GetCommonName() string {
	return self.Domain
}

func (self *SCertificate) GetSubjectAlternativeNames() string {
	return strings.Join(self.GetDetails().SubjectAltName, ",")
}

func (self *SCertificate) GetFingerprint() string {
	_fp := sha1.Sum([]byte(self.GetDetails().CertificatePublicKey))
	fp := fmt.Sprintf("sha1:% x", _fp)
	return strings.Replace(fp, " ", ":", -1)
}

func (self *SCertificate) GetExpireTime() time.Time {
	return self.CERTEndTime
}

func (self *SCertificate) GetProjectId() string {
	return self.ProjectID
}

// ssl.tencentcloudapi.com
/*
????????? 0???????????????1???????????????2??????????????????3???????????????4???????????? DNS ???????????????5???OV/EV ???????????????????????????6?????????????????????7???????????????8????????????????????? ?????????????????????
*/
func (self *SRegion) GetCertificates(projectId, certificateStatus, searchKey string) ([]SCertificate, error) {
	params := map[string]string{}
	params["Limit"] = "100"
	if len(projectId) > 0 {
		params["ProjectId"] = projectId
	}

	if len(certificateStatus) > 0 {
		params["CertificateStatus.0"] = certificateStatus
	}

	if len(searchKey) > 0 {
		params["SearchKey"] = searchKey
	}

	certs := []SCertificate{}
	offset := 0
	total := 100
	for total > offset {
		params["Offset"] = strconv.Itoa(offset)
		resp, err := self.sslRequest("DescribeCertificates", params)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeCertificates")
		}

		_certs := []SCertificate{}
		err = resp.Unmarshal(&certs, "Certificates")
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal.Certificates")
		}

		err = resp.Unmarshal(&total, "TotalCount")
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal.TotalCount")
		}

		certs = append(certs, _certs...)
		offset += 100
	}

	for i := range certs {
		certs[i].region = self
	}

	return certs, nil
}

// https://cloud.tencent.com/document/product/400/41674
func (self *SRegion) GetCertificate(certId string) (*SCertificate, error) {
	params := map[string]string{
		"CertificateId": certId,
	}

	resp, err := self.sslRequest("DescribeCertificateDetail", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeCertificateDetail")
	}

	cert := &SCertificate{}
	err = resp.Unmarshal(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	cert.region = self

	return cert, nil
}

// https://cloud.tencent.com/document/product/400/41665
// ????????????ID
func (self *SRegion) CreateCertificate(projectId, publicKey, privateKey, certType, desc string) (string, error) {
	params := map[string]string{
		"CertificatePublicKey": publicKey,
		"CertificateType":      certType,
		"Alias":                desc,
	}

	if len(privateKey) > 0 {
		params["CertificatePrivateKey"] = privateKey
	} else {
		if certType == "SVR" {
			return "", fmt.Errorf("certificate private key required while certificate type is SVR")
		}
	}

	if len(projectId) > 0 {
		params["ProjectId"] = projectId
	}

	resp, err := self.sslRequest("UploadCertificate", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("CertificateId")
}

// https://cloud.tencent.com/document/product/400/41675
func (self *SRegion) DeleteCertificate(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("DelteCertificate certificate id should not be empty")
	}

	params := map[string]string{"CertificateId": id}
	resp, err := self.sslRequest("DeleteCertificate", params)
	if err != nil {
		return errors.Wrap(err, "DeleteCertificate")
	}

	if deleted, _ := resp.Bool("DeleteResult"); deleted {
		return nil
	} else {
		return fmt.Errorf("DeleteCertificate %s", resp)
	}
}
