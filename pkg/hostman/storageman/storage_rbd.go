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

// +build linux,cgo

package storageman

import (
	"context"
	"fmt"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/cephutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

const (
	RBD_FEATURE = 3
	RBD_ORDER   = 22 //为rbd对应到rados中每个对象的大小，默认为4MB
)

type sStorageConf struct {
	MonHost            string
	Key                string
	Pool               string
	RadosMonOpTimeout  int64
	RadosOsdOpTimeout  int64
	ClientMountTimeout int64
}

type SRbdStorage struct {
	SBaseStorage
	sStorageConf
}

func NewRBDStorage(manager *SStorageManager, path string) *SRbdStorage {
	var ret = new(SRbdStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, path)
	ret.sStorageConf = sStorageConf{}
	return ret
}

type SRbdStorageFactory struct {
}

func (factory *SRbdStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewRBDStorage(manager, mountPoint)
}

func (factory *SRbdStorageFactory) StorageType() string {
	return api.STORAGE_RBD
}

func init() {
	registerStorageFactory(&SRbdStorageFactory{})
}

func (s *SRbdStorage) StorageType() string {
	return api.STORAGE_RBD
}

func (s *SRbdStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return ""
}

func (s *SRbdStorage) GetClient() (*cephutils.CephClient, error) {
	return cephutils.NewClient(s.MonHost, s.Key, s.Pool)
}

func (s *SRbdStorage) IsSnapshotExist(diskId, snapshotId string) (bool, error) {
	client, err := s.GetClient()
	if err != nil {
		return false, errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	img, err := client.GetImage(diskId)
	if err != nil {
		return false, errors.Wrapf(err, "GetImage")
	}
	return img.IsSnapshotExist(snapshotId)
}

func (s *SRbdStorage) GetSnapshotDir() string {
	return ""
}

func (s *SRbdStorage) GetFuseTmpPath() string {
	return ""
}

func (s *SRbdStorage) GetFuseMountPath() string {
	return ""
}

func (s *SRbdStorage) GetImgsaveBackupPath() string {
	return ""
}

//Tip Configuration values containing :, @, or = can be escaped with a leading \ character.
func (s *SRbdStorage) getStorageConfString() string {
	conf := []string{}
	conf = append(conf, "mon_host="+strings.ReplaceAll(s.MonHost, ",", `\;`))
	key := s.Key
	if len(key) > 0 {
		for _, k := range []string{":", "@", "="} {
			key = strings.ReplaceAll(key, k, fmt.Sprintf(`\%s`, k))
		}
		conf = append(conf, "key="+key)
	}
	for k, timeout := range map[string]int64{
		"rados_mon_op_timeout": s.RadosMonOpTimeout,
		"rados_osd_op_timeout": s.RadosOsdOpTimeout,
		"client_mount_timeout": s.ClientMountTimeout,
	} {
		conf = append(conf, fmt.Sprintf("%s=%d", k, timeout))
	}
	return ":" + strings.Join(conf, ":")
}

func (s *SRbdStorage) listImages(pool string) ([]string, error) {
	client, err := s.GetClient()
	if err != nil {
		return nil, errors.Wrapf(err, "GetClient")
	}
	client.SetPool(pool)
	defer client.Close()
	return client.ListImages()
}

func (s *SRbdStorage) IsImageExist(name string) (bool, error) {
	images, err := s.listImages(s.Pool)
	if err != nil {
		return false, errors.Wrapf(err, "listImages")
	}
	if utils.IsInStringArray(name, images) {
		return true, nil
	}
	return false, nil
}

func (s *SRbdStorage) getImageSizeMb(pool string, name string) (uint64, error) {
	client, err := s.GetClient()
	if err != nil {
		return 0, errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	client.SetPool(pool)
	img, err := client.GetImage(name)
	if err != nil {
		return 0, errors.Wrapf(err, "GetImage")
	}
	info, err := img.GetInfo()
	if err != nil {
		return 0, errors.Wrapf(err, "GetInfo")
	}
	return uint64(info.SizeByte) / 1024 / 1024, nil
}

func (s *SRbdStorage) resizeImage(pool string, name string, sizeMb uint64) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	client.SetPool(pool)
	img, err := client.GetImage(name)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	return img.Resize(int64(sizeMb))
}

func (s *SRbdStorage) deleteImage(pool string, name string) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	client.SetPool(pool)
	img, err := client.GetImage(name)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "GetImage")
	}
	return img.Delete()
}

// 速度快
func (s *SRbdStorage) cloneImage(ctx context.Context, srcPool string, srcImage string, destPool string, destImage string) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	client.SetPool(srcPool)
	img, err := client.GetImage(srcImage)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	return img.Clone(ctx, destPool, destImage)
}

func (s *SRbdStorage) cloneFromSnapshot(srcImage, srcPool, srcSnapshot, newImage, pool string) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	client.SetPool(srcPool)
	img, err := client.GetImage(srcImage)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	snap, err := img.GetSnapshot(srcSnapshot)
	if err != nil {
		return errors.Wrapf(err, "GetSnapshot")
	}
	return snap.Clone(pool, newImage)
}

func (s *SRbdStorage) createImage(pool string, name string, sizeMb uint64) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	client.SetPool(pool)
	_, err = client.CreateImage(name, int64(sizeMb))
	return err
}

func (s *SRbdStorage) renameImage(pool string, src string, dest string) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	client.SetPool(pool)
	img, err := client.GetImage(src)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	return img.Rename(dest)
}

func (s *SRbdStorage) resetDisk(pool string, diskId string, snapshotId string) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	client.SetPool(pool)
	defer client.Close()
	img, err := client.GetImage(diskId)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	snap, err := img.GetSnapshot(snapshotId)
	if err != nil {
		return errors.Wrapf(err, "GetSnapshot")
	}
	return snap.Rollback()
}

func (s *SRbdStorage) createSnapshot(pool string, diskId string, snapshotId string) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	client.SetPool(pool)
	defer client.Close()
	img, err := client.GetImage(diskId)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	_, err = img.CreateSnapshot(snapshotId)
	return err
}

func (s *SRbdStorage) deleteSnapshot(pool string, diskId string, snapshotId string) error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	client.SetPool(pool)
	defer client.Close()
	img, err := client.GetImage(diskId)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	snap, err := img.GetSnapshot(snapshotId)
	if err != nil {
		return errors.Wrapf(err, "GetSnapshot")
	}
	return snap.Delete()
}

func (s *SRbdStorage) SyncStorageSize() error {
	content := jsonutils.NewDict()
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	capacity, err := client.GetCapacity()
	if err != nil {
		return errors.Wrapf(err, "GetCapacity")
	}
	content.Set("capacity", jsonutils.NewInt(int64(capacity.CapacitySizeKb/1024)))
	content.Set("actual_capacity_used", jsonutils.NewInt(int64(capacity.UsedCapacitySizeKb/1024)))
	_, err = modules.Storages.Put(
		hostutils.GetComputeSession(context.Background()),
		s.StorageId, content)
	return errors.Wrapf(err, "storage update")
}

func (s *SRbdStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := map[string]interface{}{}
	if len(s.StorageId) > 0 {
		client, err := s.GetClient()
		if err != nil {
			return modules.Storages.PerformAction(hostutils.GetComputeSession(context.Background()), s.StorageId, "offline", nil)
		}
		defer client.Close()
		capacity, err := client.GetCapacity()
		if err != nil {
			return modules.Storages.PerformAction(hostutils.GetComputeSession(context.Background()), s.StorageId, "offline", nil)
		}

		content = map[string]interface{}{
			"name":                 s.StorageName,
			"capacity":             capacity.CapacitySizeKb / 1024,
			"actual_capacity_used": capacity.UsedCapacitySizeKb / 1024,
			"status":               api.STORAGE_ONLINE,
			"zone":                 s.GetZoneName(),
		}
		return modules.Storages.Put(hostutils.GetComputeSession(context.Background()), s.StorageId, jsonutils.Marshal(content))
	}
	return modules.Storages.Get(hostutils.GetComputeSession(context.Background()), s.StorageName, jsonutils.Marshal(content))
}

func (s *SRbdStorage) GetDiskById(diskId string) (IDisk, error) {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == diskId {
			err := s.Disks[i].Probe()
			if err != nil {
				return nil, errors.Wrapf(err, "disk.Prob")
			}
			return s.Disks[i], nil
		}
	}
	var disk = NewRBDDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (s *SRbdStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewRBDDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SRbdStorage) Accessible() error {
	client, err := s.GetClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	_, err = client.GetCapacity()
	return err
}

func (s *SRbdStorage) Detach() error {
	return nil
}

func (s *SRbdStorage) SaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	data, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	rbdImageCache := storageManager.GetStoragecacheById(s.GetStoragecacheId())
	if rbdImageCache == nil {
		return nil, fmt.Errorf("failed to find storage image cache for storage %s", s.GetStorageName())
	}

	imagePath, _ := data.GetString("image_path")
	compress := jsonutils.QueryBoolean(data, "compress", true)
	format, _ := data.GetString("format")
	imageId, _ := data.GetString("image_id")
	imageName := "image_cache_" + imageId
	if err := s.renameImage(rbdImageCache.GetPath(), imagePath, imageName); err != nil {
		return nil, err
	}

	imagePath = fmt.Sprintf("rbd:%s/%s%s", rbdImageCache.GetPath(), imageName, s.getStorageConfString())

	if err := s.saveToGlance(ctx, imageId, imagePath, compress, format); err != nil {
		log.Errorf("Save to glance failed: %s", err)
		s.onSaveToGlanceFailed(ctx, imageId)
	}

	rbdImageCache.LoadImageCache(imageId)
	_, err := hostutils.RemoteStoragecacheCacheImage(ctx, rbdImageCache.GetId(), imageId, "ready", imagePath)
	if err != nil {
		log.Errorf("Fail to remote cache image: %v", err)
	}
	return nil, nil
}

func (s *SRbdStorage) onSaveToGlanceFailed(ctx context.Context, imageId string) {
	params := jsonutils.NewDict()
	params.Set("status", jsonutils.NewString("killed"))
	_, err := modules.Images.Update(hostutils.GetImageSession(ctx, s.GetZoneName()),
		imageId, params)
	if err != nil {
		log.Errorln(err)
	}
}

func (s *SRbdStorage) saveToGlance(ctx context.Context, imageId, imagePath string, compress bool, format string) error {
	ret, err := deployclient.GetDeployClient().SaveToGlance(context.Background(),
		&deployapi.SaveToGlanceParams{DiskPath: imagePath, Compress: compress})
	if err != nil {
		return err
	}

	tmpImageFile := fmt.Sprintf("/tmp/%s.img", imageId)
	if len(format) == 0 {
		format = options.HostOptions.DefaultImageSaveFormat
	}

	err = procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuImg(),
		"convert", "-f", "raw", "-O", format, imagePath, tmpImageFile).Run()
	if err != nil {
		return err
	}

	f, err := os.Open(tmpImageFile)
	if err != nil {
		return err
	}
	defer os.Remove(tmpImageFile)
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		return err
	}
	size := finfo.Size()

	var params = jsonutils.NewDict()
	if len(ret.OsInfo) > 0 {
		params.Set("os_type", jsonutils.NewString(ret.OsInfo))
	}
	relInfo := ret.ReleaseInfo
	if relInfo != nil {
		params.Set("os_distribution", jsonutils.NewString(relInfo.Distro))
		if len(relInfo.Version) > 0 {
			params.Set("os_version", jsonutils.NewString(relInfo.Version))
		}
		if len(relInfo.Arch) > 0 {
			params.Set("os_arch", jsonutils.NewString(relInfo.Arch))
		}
		if len(relInfo.Version) > 0 {
			params.Set("os_language", jsonutils.NewString(relInfo.Language))
		}
	}
	params.Set("image_id", jsonutils.NewString(imageId))

	_, err = modules.Images.Upload(hostutils.GetImageSession(ctx, s.GetZoneName()),
		params, f, size)
	return err
}

func (s *SRbdStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return fmt.Errorf("Not support")
}

func (s *SRbdStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not support delete snapshots")
}

func (s *SRbdStorage) CreateDiskFromSnapshot(
	ctx context.Context, disk IDisk, createParams *SDiskCreateByDiskinfo,
) error {
	var (
		snapshotUrl, _ = createParams.DiskInfo.GetString("snapshot_url")
		srcDiskId, _   = createParams.DiskInfo.GetString("src_disk_id")
		srcPool, _     = createParams.DiskInfo.GetString("src_pool")
	)
	return disk.CreateFromRbdSnapshot(ctx, snapshotUrl, srcDiskId, srcPool)
}

func (s *SRbdStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if gotypes.IsNil(conf) {
		return fmt.Errorf("empty storage conf for storage %s(%s)", storageName, storageId)
	}
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		s.StorageConf = dconf
	}
	conf.Unmarshal(&s.sStorageConf)
	if s.RadosMonOpTimeout == 0 {
		s.RadosMonOpTimeout = api.RBD_DEFAULT_MON_TIMEOUT
	}
	if s.RadosOsdOpTimeout == 0 {
		s.RadosOsdOpTimeout = api.RBD_DEFAULT_OSD_TIMEOUT
	}
	if s.ClientMountTimeout == 0 {
		s.ClientMountTimeout = api.RBD_DEFAULT_MOUNT_TIMEOUT
	}
	return nil
}