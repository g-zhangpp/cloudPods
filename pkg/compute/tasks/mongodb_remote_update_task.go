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

package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type MongoDBRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(MongoDBRemoteUpdateTask{})
}

func (self *MongoDBRemoteUpdateTask) taskFail(ctx context.Context, mongodb *models.SMongoDB, err error) {
	mongodb.SetStatus(self.UserCred, apis.STATUS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *MongoDBRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	mongodb := obj.(*models.SMongoDB)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	iMongoDB, err := mongodb.GetIMongoDB()
	if err != nil {
		self.taskFail(ctx, mongodb, errors.Wrapf(err, "GetIMongoDB"))
		return
	}

	oldTags, err := iMongoDB.GetTags()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, mongodb, nil)
			return
		}
		self.taskFail(ctx, mongodb, errors.Wrapf(err, "GetTags"))
		return
	}
	tags, err := mongodb.GetAllUserMetadata()
	if err != nil {
		self.taskFail(ctx, mongodb, errors.Wrapf(err, "GetAllUserMetadata"))
		return
	}
	tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
	err = cloudprovider.SetTags(ctx, iMongoDB, mongodb.ManagerId, tags, replaceTags)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, mongodb, nil)
			return
		}
		logclient.AddActionLogWithStartable(self, mongodb, logclient.ACT_UPDATE_TAGS, err, self.GetUserCred(), false)
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	logclient.AddActionLogWithStartable(self, mongodb, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, self.GetUserCred(), true)
	self.OnRemoteUpdateComplete(ctx, mongodb, nil)
}

func (self *MongoDBRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, mongodb *models.SMongoDB, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, mongodb, "MongoDBSyncstatusTask", self.GetTaskId())
}

func (self *MongoDBRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, mongodb *models.SMongoDB, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *MongoDBRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, mongodb *models.SMongoDB, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
