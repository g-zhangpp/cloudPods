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

package apsara

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	v "yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_APSARA    = api.CLOUD_PROVIDER_APSARA
	CLOUD_PROVIDER_APSARA_CN = "阿里云专有云"
	CLOUD_PROVIDER_APSARA_EN = "Aliyun Apsara"

	APSARA_API_VERSION     = "2014-05-26"
	APSARA_API_VERSION_VPC = "2016-04-28"
	APSARA_API_VERSION_LB  = "2014-05-15"
	APSARA_API_VERSION_KVS = "2015-01-01"

	APSARA_API_VERSION_TRIAL = "2017-12-04"

	APSARA_BSS_API_VERSION = "2017-12-14"

	APSARA_RAM_API_VERSION  = "2015-05-01"
	APSARA_API_VERION_RDS   = "2014-08-15"
	APSARA_ASCM_API_VERSION = "2019-05-10"
	APSARA_STS_API_VERSION  = "2015-04-01"

	APSARA_PRODUCT_METRICS      = "Cms"
	APSARA_PRODUCT_RDS          = "Rds"
	APSARA_PRODUCT_VPC          = "Vpc"
	APSARA_PRODUCT_KVSTORE      = "R-kvstore"
	APSARA_PRODUCT_SLB          = "Slb"
	APSARA_PRODUCT_ECS          = "Ecs"
	APSARA_PRODUCT_ACTION_TRIAL = "actiontrail"
	APSARA_PRODUCT_STS          = "Sts"
	APSARA_PRODUCT_RAM          = "Ram"
	APSARA_PRODUCT_ASCM         = "ascm"
)

type ApsaraClientConfig struct {
	cpcfg        cloudprovider.ProviderConfig
	accessKey    string
	accessSecret string
	debug        bool

	endpoints cloudprovider.SApsaraEndpoints
}

func NewApsaraClientConfig(accessKey, accessSecret string, endpoint string, endpoints cloudprovider.SApsaraEndpoints) *ApsaraClientConfig {
	cfg := &ApsaraClientConfig{
		accessKey:    accessKey,
		accessSecret: accessSecret,
		endpoints:    endpoints,
	}
	cfg.cpcfg.URL = endpoint
	return cfg
}

func (cfg *ApsaraClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *ApsaraClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *ApsaraClientConfig) Debug(debug bool) *ApsaraClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg ApsaraClientConfig) Copy() ApsaraClientConfig {
	return cfg
}

type SApsaraClient struct {
	*ApsaraClientConfig

	ownerId   string
	ownerName string

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket
}

func NewApsaraClient(cfg *ApsaraClientConfig) (*SApsaraClient, error) {
	client := SApsaraClient{
		ApsaraClientConfig: cfg,
	}

	err := client.fetchRegions()
	if err != nil {
		return nil, errors.Wrap(err, "fetchRegions")
	}
	if len(client.endpoints.OssEndpoint) > 0 {
		err = client.fetchBuckets()
		if err != nil {
			return nil, errors.Wrapf(err, "fetchBuckets")
		}
		if client.debug {
			log.Debugf("ClientID: %s ClientName: %s", client.ownerId, client.ownerName)
		}
	}
	return &client, nil
}

func (self *SApsaraClient) getDomain(product string) string {
	switch product {
	case APSARA_PRODUCT_ECS:
		if len(self.endpoints.EcsEndpoint) > 0 {
			return self.endpoints.EcsEndpoint
		}
	case APSARA_PRODUCT_RAM:
		if len(self.endpoints.RamEndpoint) > 0 {
			return self.endpoints.RamEndpoint
		}
	case APSARA_PRODUCT_RDS:
		if len(self.endpoints.RdsEndpoint) > 0 {
			return self.endpoints.RdsEndpoint
		}
	case APSARA_PRODUCT_SLB:
		if len(self.endpoints.SlbEndpoint) > 0 {
			return self.endpoints.SlbEndpoint
		}
	case APSARA_PRODUCT_STS:
		if len(self.endpoints.StsEndpoint) > 0 {
			return self.endpoints.StsEndpoint
		}
	case APSARA_PRODUCT_VPC:
		if len(self.endpoints.VpcEndpoint) > 0 {
			return self.endpoints.VpcEndpoint
		}
	case APSARA_PRODUCT_KVSTORE:
		if len(self.endpoints.KvsEndpoint) > 0 {
			return self.endpoints.KvsEndpoint
		}
	}
	return self.cpcfg.URL
}

func productRequest(client *sdk.Client, product, domain, apiVersion, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	params["Product"] = product
	return jsonRequest(client, domain, apiVersion, apiName, params, debug)
}

func jsonRequest(client *sdk.Client, domain, apiVersion, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	if debug {
		log.Debugf("request %s %s %s %s", domain, apiVersion, apiName, params)
	}
	for i := 1; i < 4; i++ {
		resp, err := _jsonRequest(client, domain, apiVersion, apiName, params)
		retry := false
		if err != nil {
			for _, code := range []string{
				"InvalidAccessKeyId.NotFound",
			} {
				if strings.Contains(err.Error(), code) {
					return nil, err
				}
			}
			for _, code := range []string{"404 Not Found", "EntityNotExist.Role", "EntityNotExist.Group"} {
				if strings.Contains(err.Error(), code) {
					return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
				}
			}
			for _, code := range []string{
				"EOF",
				"i/o timeout",
				"TLS handshake timeout",
				"connection reset by peer",
				"server misbehaving",
				"SignatureNonceUsed",
				"InvalidInstance.NotSupported",
				"try later",
				"BackendServer.configuring",
				"Another operation is being performed", //Another operation is being performed on the DB instance or the DB instance is faulty(赋予RDS账号权限)
			} {
				if strings.Contains(err.Error(), code) {
					retry = true
					break
				}
			}
		}
		if retry {
			if debug {
				log.Debugf("Retry %d...", i)
			}
			time.Sleep(time.Second * time.Duration(i*10))
			continue
		}
		if debug {
			log.Debugf("Response: %s", resp)
		}
		return resp, err
	}
	return nil, fmt.Errorf("timeout for request %s params: %s", apiName, params)
}

func _jsonRequest(client *sdk.Client, domain string, version string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	req := requests.NewCommonRequest()
	req.Domain = domain
	req.Version = version
	req.ApiName = apiName
	id := ""
	if params != nil {
		for k, v := range params {
			req.QueryParams[k] = v
			if strings.ToLower(k) != "regionid" && strings.HasSuffix(k, "Id") {
				id = v
			}
		}
	}
	req.Scheme = "http"
	req.GetHeaders()["User-Agent"] = "vendor/yunion-OneCloud@" + v.Get().GitVersion
	if strings.HasPrefix(apiName, "Describe") && len(id) > 0 {
		req.GetHeaders()["x-acs-instanceId"] = id
	}

	resp, err := processCommonRequest(client, req)
	if err != nil {
		return nil, errors.Wrapf(err, "processCommonRequest(%s, %s)", apiName, params)
	}
	body, err := jsonutils.Parse(resp.GetHttpContentBytes())
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Parse")
	}
	//{"Code":"InvalidInstanceType.ValueNotSupported","HostId":"ecs.apsaracs.com","Message":"The specified instanceType beyond the permitted range.","RequestId":"0042EE30-0EDF-48A7-A414-56229D4AD532"}
	//{"Code":"200","Message":"successful","PageNumber":1,"PageSize":50,"RequestId":"BB4C970C-0E23-48DC-A3B0-EB21FFC70A29","RouterTableList":{"RouterTableListType":[{"CreationTime":"2017-03-19T13:37:40Z","Description":"","ResourceGroupId":"rg-acfmwie3cqoobmi","RouteTableId":"vtb-j6c60lectdi80rk5xz43g","RouteTableName":"","RouteTableType":"System","RouterId":"vrt-j6c00qrol733dg36iq4qj","RouterType":"VRouter","VSwitchIds":{"VSwitchId":["vsw-j6c3gig5ub4fmi2veyrus"]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}]},"Success":true,"TotalCount":1}
	if body.Contains("Code") {
		code, _ := body.GetString("Code")
		if len(code) > 0 && !utils.IsInStringArray(code, []string{"200"}) {
			return nil, fmt.Errorf(body.String())
		}
	}
	if body.Contains("errorKey") {
		return nil, errors.Errorf(body.String())
	}
	return body, nil
}

func (self *SApsaraClient) getDefaultClient() (*sdk.Client, error) {
	regionId := ""
	if len(self.iregions) > 0 {
		regionId = self.iregions[0].GetId()
	}
	transport := httputils.GetTransport(true)
	transport.Proxy = self.cpcfg.ProxyFunc
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client, err := sdk.NewClientWithOptions(
		regionId,
		&sdk.Config{
			HttpTransport: transport,
		},
		&credentials.BaseCredential{
			AccessKeyId:     self.accessKey,
			AccessKeySecret: self.accessSecret,
		},
	)
	return client, err
}

func (self *SApsaraClient) ascmRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return productRequest(cli, APSARA_PRODUCT_ASCM, self.cpcfg.URL, APSARA_ASCM_API_VERSION, apiName, params, self.debug)
}

func (self *SApsaraClient) ecsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	domain := self.getDomain(APSARA_PRODUCT_ECS)
	return productRequest(cli, APSARA_PRODUCT_ECS, domain, APSARA_API_VERSION, apiName, params, self.debug)
}

func (self *SApsaraClient) trialRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	domain := self.getDomain(APSARA_PRODUCT_ACTION_TRIAL)
	return productRequest(cli, APSARA_PRODUCT_ACTION_TRIAL, domain, APSARA_API_VERSION_TRIAL, apiName, params, self.debug)
}

func (self *SApsaraClient) fetchRegions() error {
	params := map[string]string{"AcceptLanguage": "zh-CN"}
	if len(self.endpoints.DefaultRegion) > 0 {
		params["RegionId"] = self.endpoints.DefaultRegion
	}
	body, err := self.ecsRequest("DescribeRegions", params)
	if err != nil {
		return errors.Wrapf(err, "DescribeRegions")
	}

	regions := make([]SRegion, 0)
	err = body.Unmarshal(&regions, "Regions", "Region")
	if err != nil {
		return errors.Wrapf(err, "body.Unmarshal")
	}
	self.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i += 1 {
		regions[i].client = self
		self.iregions[i] = &regions[i]
	}
	return nil
}

// https://help.apsara.com/document_detail/31837.html?spm=a2c4g.11186623.2.6.XqEgD1
func (client *SApsaraClient) getOssClient(regionId string) (*oss.Client, error) {
	// NOTE
	//
	// oss package as of version 20181116160301-c6838fdc33ed does not
	// respect http.ProxyFromEnvironment.
	//
	// The ClientOption Proxy, AuthProxy lacks the feature NO_PROXY has
	// which can be used to whitelist ips, domains from http_proxy,
	// https_proxy setting
	// oss use no timeout client so as to send/download large files
	httpClient := client.cpcfg.AdaptiveTimeoutHttpClient()
	cliOpts := []oss.ClientOption{
		oss.HTTPClient(httpClient),
	}
	cli, err := oss.New(client.endpoints.OssEndpoint, client.accessKey, client.accessSecret, cliOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "oss.New")
	}
	return cli, nil
}

func (self *SApsaraClient) getRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SApsaraClient) invalidateIBuckets() {
	self.iBuckets = nil
}

func (self *SApsaraClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if len(self.endpoints.OssEndpoint) == 0 {
		return nil, fmt.Errorf("empty oss endpoint")
	}
	if self.iBuckets == nil {
		err := self.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.iBuckets, nil
}

func (self *SApsaraClient) fetchBuckets() error {
	osscli, err := self.getOssClient("")
	if err != nil {
		return errors.Wrap(err, "self.getOssClient")
	}
	result, err := osscli.ListBuckets()
	if err != nil {
		return errors.Wrap(err, "oss.ListBuckets")
	}

	self.ownerId = result.Owner.ID
	self.ownerName = result.Owner.DisplayName

	ret := make([]cloudprovider.ICloudBucket, 0)
	for _, bInfo := range result.Buckets {
		regionId := bInfo.Location
		if strings.HasPrefix(regionId, "oss-") {
			regionId = regionId[4:]
		}
		region, err := self.getRegionByRegionId(regionId)
		if err != nil {
			log.Errorf("cannot find bucket %s region %s", bInfo.Name, regionId)
			continue
		}
		b := SBucket{
			region:       region.(*SRegion),
			Name:         bInfo.Name,
			Location:     bInfo.Location,
			CreationDate: bInfo.CreationDate,
			StorageClass: bInfo.StorageClass,
		}
		ret = append(ret, &b)
	}
	self.iBuckets = ret
	return nil
}

func (self *SApsaraClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SApsaraClient) GetProvider() string {
	return self.cpcfg.Vendor
}

func (self *SApsaraClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	err := self.fetchRegions()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SApsaraClient) GetAccountId() string {
	return self.cpcfg.URL
}

func (self *SApsaraClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SApsaraClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SApsaraClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SApsaraClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SApsaraClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIVpcById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SApsaraClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SApsaraClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	pageSize, pageNumber := 50, 1
	resourceGroups := []SResourceGroup{}
	for {
		parts, total, err := self.GetResourceGroups(pageNumber, pageSize)
		if err != nil {
			return nil, errors.Wrap(err, "GetResourceGroups")
		}
		resourceGroups = append(resourceGroups, parts...)
		if len(resourceGroups) >= total {
			break
		}
		pageNumber += 1
	}
	ret := []cloudprovider.ICloudProject{}
	for i := range resourceGroups {
		ret = append(ret, &resourceGroups[i])
	}
	return ret, nil
}

func (region *SApsaraClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
	}
	return caps
}
