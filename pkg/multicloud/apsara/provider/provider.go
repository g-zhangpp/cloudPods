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

package provider

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/apsara"
)

type SApsaraProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SApsaraProviderFactory) GetId() string {
	return apsara.CLOUD_PROVIDER_APSARA
}

func (self *SApsaraProviderFactory) GetName() string {
	return apsara.CLOUD_PROVIDER_APSARA_CN
}

func (self *SApsaraProviderFactory) IsMultiTenant() bool {
	return true
}

func (self *SApsaraProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	if len(input.Endpoint) == 0 {
		return output, httperrors.NewMissingParameterError("endpoint")
	}
	output.AccessUrl = input.Endpoint
	if input.SApsaraEndpoints == nil || len(input.SApsaraEndpoints.DefaultRegion) == 0 {
		return output, httperrors.NewMissingParameterError("default_region")
	}
	return output, nil
}

func (self *SApsaraProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}
	output = cloudprovider.SCloudaccount{
		Account: input.AccessKeyId,
		Secret:  input.AccessKeySecret,
	}
	return output, nil
}

func (self *SApsaraProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	endpoints := cloudprovider.SApsaraEndpoints{}
	if cfg.Options != nil {
		cfg.Options.Unmarshal(&endpoints)
	}
	client, err := apsara.NewApsaraClient(
		apsara.NewApsaraClientConfig(
			cfg.Account,
			cfg.Secret,
			cfg.URL,
			endpoints,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SApsaraProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SApsaraProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"APSARA_ACCESS_KEY": info.Account,
		"APSARA_SECRET":     info.Secret,
		"APSARA_ENDPOINT":   info.Url,
		"APSARA_REGION":     "",
	}, nil
}

func init() {
	factory := SApsaraProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SApsaraProvider struct {
	cloudprovider.SBaseProvider
	client *apsara.SApsaraClient
}

func (self *SApsaraProvider) WithClient(client *apsara.SApsaraClient) *SApsaraProvider {
	self.client = client
	return self
}

func (self *SApsaraProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(apsara.APSARA_API_VERSION), "api_version")
	return info, nil
}

func (self *SApsaraProvider) GetVersion() string {
	return apsara.APSARA_API_VERSION
}

func (self *SApsaraProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SApsaraProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SApsaraProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SApsaraProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SApsaraProvider) GetBalance() (float64, string, error) {
	return 0, api.CLOUD_PROVIDER_HEALTH_NORMAL, nil
}

func (self *SApsaraProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SApsaraProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return self.client.CreateIProject(name)
}

func (self *SApsaraProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"Standard", "IA", "Archive",
	}
}

func (self *SApsaraProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SApsaraProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SApsaraProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SApsaraProvider) GetIamLoginUrl() string {
	return self.client.GetIamLoginUrl()
}

func (self *SApsaraProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SApsaraProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SApsaraProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroups()
}

func (self *SApsaraProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroupByName(name)
}

func (self *SApsaraProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SApsaraProvider) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.client.CreateICloudgroup(name, desc)
}

func (self *SApsaraProvider) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetISystemCloudpolicies()
}

func (self *SApsaraProvider) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetICustomCloudpolicies()
}

func (self *SApsaraProvider) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	return self.client.CreateICloudpolicy(opts)
}
