// Design for copy data from source PVC to destination PVC, continuously push statue into status channel for notifications
package datacopy

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/hwameistor/hwameistor/pkg/exechelper"
	"github.com/hwameistor/hwameistor/pkg/exechelper/nsexecutor"
)

const (
	DefaultCopyTimeout     = time.Hour * 48
	syncMountContainerName = "syncer"

	SyncKeyDir             = "/root/.ssh"
	SyncKeyComment         = "SyncPubKey"
	sshkeygenCmd           = "ssh-keygen"
	SyncPubKeyFileName     = "sync.pub"
	SyncPrivateKeyFileName = "sync"
	SyncKeyConfigMapName   = "sync-key-config"
	SyncCertKey            = "sync-ssh-keys"
)

var (
	logger = log.WithField("module", "util-job")
)

type DataCopyManager struct {
	dataCopyJobStatusAnnotationName string
	statusGenerator                 *statusGenerator
	k8sControllerClient             k8sclient.Client
	ctx                             context.Context
	//progressWatchingFunc            func() *Progress

	workingNamespace string
	syncer           DataSyncer
	cmdExec          exechelper.Executor
}

// NewDataCopyManager return DataCopyManager instance
//
// It will feedback copy process status continuously through statusCh,
// so it dose not need ResourceReady to poll resource status
func NewDataCopyManager(ctx context.Context, syncToolName string, dataCopyJobStatusAnnotationName string,
	client k8sclient.Client, statusCh chan *DataCopyStatus, namespace string) (*DataCopyManager, error) {
	dcm := &DataCopyManager{
		dataCopyJobStatusAnnotationName: dataCopyJobStatusAnnotationName,
		k8sControllerClient:             client,
		ctx:                             ctx,
		cmdExec:                         nsexecutor.New(),
		syncer:                          NewSyncer(syncToolName, namespace, client),
		workingNamespace:                namespace,
	}

	statusGenerator, err := newStatusGenerator(dcm, dataCopyJobStatusAnnotationName, statusCh, namespace)
	if err != nil {
		logger.WithError(err).Error("Failed to init StatusGenerator")
		return nil, err
	}

	dcm.statusGenerator = statusGenerator
	return dcm, nil
}

func (dcm *DataCopyManager) Run() {
	logger.Debugf("DataCopyManager Run start")
	dcm.statusGenerator.Run()
}

func (dcm *DataCopyManager) Sync(jobName, srcNodeName, dstNodeName, volName string) error {
	logCtx := logger.WithFields(log.Fields{"job": jobName, "volume": volName})
	logCtx.Debug("Preparing the resources for data sync ...")

	if err := dcm.prepareForSync(jobName, srcNodeName, dstNodeName, volName); err != nil {
		return err
	}

	ctx := context.TODO()

	cmName := GetConfigMapName(SyncConfigMapName, volName)
	cm := &corev1.ConfigMap{}
	if err := dcm.k8sControllerClient.Get(context.TODO(), types.NamespacedName{Namespace: dcm.workingNamespace, Name: cmName}, cm); err != nil {
		logCtx.WithField("configmap", cmName).Error("Not found the data sync configmap")
		return err
	}

	if ready := cm.Data[SyncConfigSourceNodeReadyKey]; ready != SyncTrue {
		logCtx.WithField(SyncConfigSourceNodeReadyKey, ready).Debug("Waiting for source mountpoint to be ready ...")
		return fmt.Errorf("source mountpoint is not ready")
	}
	if ready := cm.Data[SyncConfigRemoteNodeReadyKey]; ready != SyncTrue {
		logCtx.WithField(SyncConfigRemoteNodeReadyKey, ready).Debug("Waiting for remote mountpoint to be ready ...")
		return fmt.Errorf("remote mountpoint is not ready")
	}

	syncJob := &batchv1.Job{}
	if err := dcm.k8sControllerClient.Get(ctx, types.NamespacedName{Namespace: dcm.workingNamespace, Name: jobName}, syncJob); err != nil {
		if errors.IsNotFound(err) {
			logCtx.WithField("Job", jobName).Info("No job is created to sync replicas, create one ...")
			if err := dcm.syncer.StartSync(jobName, volName, srcNodeName, ""); err != nil {
				logCtx.WithField("LocalVolume", volName).WithError(err).Error("Failed to start a job to sync replicas")
				return fmt.Errorf("failed to start a job to sync replicas for volume %s", volName)
			}
			return fmt.Errorf("syncing replica still in progress")
		}
		logCtx.WithError(err).Error("Failed to get MigrateJob from cache")
		return err
	}

	// found the job, check the status
	isJobCompleted := false
	for _, cond := range syncJob.Status.Conditions {
		if cond.Type == batchv1.JobComplete && syncJob.Status.CompletionTime != nil && syncJob.Status.StartTime != nil {
			logCtx.WithFields(log.Fields{
				"Job":          syncJob.Name,
				"Namespace":    syncJob.Namespace,
				"StartTime":    syncJob.Status.StartTime.String(),
				"CompleteTime": syncJob.Status.CompletionTime.String(),
			}).Debug("The replicas have already been synchronized successfully")

			cm.Data[SyncConfigSyncDoneKey] = SyncTrue
			if err := dcm.k8sControllerClient.Update(ctx, cm, &k8sclient.UpdateOptions{Raw: &metav1.UpdateOptions{}}); err != nil {
				logCtx.WithField("configmap", cmName).WithError(err).Error("Failed to update rclone configmap")
				return err
			}
			// remove the finalizer will release the job
			syncJob.Finalizers = []string{}
			if err := dcm.k8sControllerClient.Update(ctx, syncJob); err != nil {
				logCtx.WithField("Job", syncJob).WithError(err).Error("Failed to remove finalizer")
				return err
			}
			if err := dcm.k8sControllerClient.Get(ctx, types.NamespacedName{Namespace: dcm.workingNamespace, Name: jobName}, syncJob); err != nil {
				if !errors.IsNotFound(err) {
					logCtx.WithField("Job", syncJob).WithError(err).Error("Failed to fetch the job")
					return err
				}
			} else {
				if err := dcm.k8sControllerClient.Delete(ctx, syncJob); err != nil {
					logCtx.WithField("Job", syncJob).WithError(err).Error("Failed to cleanup the job")
					return err
				}
			}
			if err := dcm.k8sControllerClient.Delete(ctx, cm); err != nil {
				logCtx.WithField("configmap", cm.Name).WithError(err).Warning("Failed to cleanup the rclone configmap, just leak it")
			}
			isJobCompleted = true
			break
		}
	}
	if !isJobCompleted {
		return fmt.Errorf("waiting for the sync job to complete: %s", syncJob.Name)
	}

	logCtx.Debug("Sync has already been executed successfully")
	return nil
}

func (dcm *DataCopyManager) prepareForSync(jobName, srcNodeName, dstNodeName, volName string) error {
	logCtx := logger.WithFields(log.Fields{"job": jobName, "volume": volName})
	logCtx.Debug("Preparing the resources for volume sync")

	if err := dcm.prepareRemoteAccessKeys(); err != nil {
		logCtx.WithError(err).Error("Failed to create ssh keys for rclone")
		return err
	}

	// Prepare the data syncer's configuration, which should be created unique for each volume data copy
	if err := dcm.syncer.Prepare(dstNodeName, srcNodeName, volName); err != nil {
		logCtx.WithError(err).Error("Failed to create rclone's config")
		return err
	}

	logCtx.Debug("Sync is ready to execute")

	return nil
}

func (dcm *DataCopyManager) prepareRemoteAccessKeys() error {
	ctx := context.TODO()
	// Prepare the public/private ssh keys for rclone to execute. The keys should be shared by all the rclone executions.
	// Don't update once it exists
	cm := &corev1.ConfigMap{}
	if err := dcm.k8sControllerClient.Get(ctx, types.NamespacedName{Namespace: dcm.workingNamespace, Name: SyncKeyConfigMapName}, cm); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		cm = dcm.GenerateSyncKeyConfigMap()
		if err := dcm.k8sControllerClient.Create(ctx, cm); err != nil {
			return err
		}
	}
	return nil
}

func (dcm *DataCopyManager) GenerateSyncKeyConfigMap() *corev1.ConfigMap {

	var cm = &corev1.ConfigMap{}
	syncPubKeyData, syncPrivateKeyData, err := dcm.generateSSHPubAndPrivateKeyCM()

	if err != nil {
		logger.WithError(err).Errorf("generateRcloneKeyConfigMap generateSSHPubAndPrivateKeyCM")
		return cm
	}

	configData := map[string]string{
		SyncPubKeyFileName:     syncPubKeyData,
		SyncPrivateKeyFileName: syncPrivateKeyData,
		SyncCertKey:            syncPrivateKeyData + "\n" + syncPubKeyData,
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SyncKeyConfigMapName,
			Namespace: dcm.workingNamespace,
		},
		Data: configData,
	}

	return configMap
}

func (dcm *DataCopyManager) generateSSHPubAndPrivateKeyCM() (string, string, error) {
	logger.Debug("GenerateSSHPubAndPrivateKey start ")

	keyFilePath := filepath.Join(SyncKeyDir, SyncPrivateKeyFileName)
	paramsRemove := exechelper.ExecParams{
		CmdName: "rm",
		CmdArgs: []string{"-rf", keyFilePath},
		Timeout: 0,
	}
	resultRemove := dcm.cmdExec.RunCommand(paramsRemove)
	if resultRemove.ExitCode != 0 {
		return "", "", fmt.Errorf("rm -rf %s err: %d, %s", keyFilePath, resultRemove.ExitCode, resultRemove.ErrBuf.String())
	}

	paramsMkdir := exechelper.ExecParams{
		CmdName: "mkdir",
		CmdArgs: []string{"-p", SyncKeyDir},
		Timeout: 0,
	}
	resultMkdir := dcm.cmdExec.RunCommand(paramsMkdir)
	if resultMkdir.ExitCode != 0 {
		return "", "", fmt.Errorf("mkdir -p %s err: %d, %s", SyncKeyDir, resultMkdir.ExitCode, resultMkdir.ErrBuf.String())
	}

	params := exechelper.ExecParams{
		CmdName: sshkeygenCmd,
		CmdArgs: []string{"-q", "-b 4096", "-C" + SyncKeyComment, "-f", keyFilePath},
		Timeout: 0,
	}
	result := dcm.cmdExec.RunCommand(params)
	if result.ExitCode != 0 {
		return "", "", fmt.Errorf("ssh-keygen %s err: %d, %s", SyncKeyComment, result.ExitCode, result.ErrBuf.String())
	}

	paramsCatRclone := exechelper.ExecParams{
		CmdName: "cat",
		CmdArgs: []string{keyFilePath},
		Timeout: 0,
	}
	resultCatRclone := dcm.cmdExec.RunCommand(paramsCatRclone)
	if resultCatRclone.ExitCode != 0 {
		return "", "", fmt.Errorf("cat %s err: %d, %s", keyFilePath, resultCatRclone.ExitCode, resultCatRclone.ErrBuf.String())
	}

	paramsCatRclonePub := exechelper.ExecParams{
		CmdName: "cat",
		CmdArgs: []string{keyFilePath},
		Timeout: 0,
	}
	resultCatRclonePub := dcm.cmdExec.RunCommand(paramsCatRclonePub)
	if resultCatRclonePub.ExitCode != 0 {
		return "", "", fmt.Errorf("cat %s err: %d, %s", keyFilePath, resultCatRclonePub.ExitCode, resultCatRclonePub.ErrBuf.String())
	}
	PubKeyData := resultCatRclonePub.OutBuf.String()
	PrivateKeyData := resultCatRclone.OutBuf.String()

	return PubKeyData, PrivateKeyData, nil
}

func (dcm *DataCopyManager) RegisterRelatedJob(jobName string, resultCh chan *DataCopyStatus) {
	dcm.statusGenerator.relatedJobWithResultCh[jobName] = resultCh
}

func (dcm *DataCopyManager) DeregisterRelatedJob(jobName string) {
	delete(dcm.statusGenerator.relatedJobWithResultCh, jobName)
}
