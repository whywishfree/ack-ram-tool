package credentialplugin

import (
	"context"
	"fmt"
	"github.com/AliyunContainerService/ack-ram-tool/pkg/ctl/common"
	"github.com/AliyunContainerService/ack-ram-tool/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	versionV1           = "v1"
	versionV1beta1      = "v1beta1"
	groupVersionV1      = "client.authentication.k8s.io/v1"
	groupVersionV1beta1 = "client.authentication.k8s.io/v1beta1"
	kindExecCredential  = "ExecCredential"

	commandName = "ack-ram-tool"
)

var getKubeconfigCmd = &cobra.Command{
	Use:   "get-kubeconfig",
	Short: "get a kubeconfig with exec credential plugin format.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		client := common.GetClientOrDie()
		ctx := context.Background()

		kubeconfig, err := client.GetUserKubeConfig(ctx, getCredentialOpts.clusterId,
			getCredentialOpts.privateIpAddress, getCredentialOpts.temporaryDuration)
		common.ExitIfError(err)
		newConf := generateExecKubeconfig(kubeconfig)

		d, err := yaml.Marshal(newConf)
		common.ExitIfError(err)
		fmt.Println(string(d))
	},
}

func generateExecKubeconfig(config *types.KubeConfig) *types.KubeConfig {
	newConf := &types.KubeConfig{
		Kind:           config.Kind,
		APIVersion:     config.APIVersion,
		Clusters:       config.Clusters,
		Contexts:       config.Contexts,
		CurrentContext: config.CurrentContext,
		Users:          config.Users,
		Preferences:    config.Preferences,
	}
	var users []types.KubeAuthUser
	args := []string{
		"credential-plugin",
		"get-credential",
		"--cluster-id",
		getCredentialOpts.clusterId,
		//"--api-version",
		//getCredentialOpts.apiVersion,
		//"--expiration",
		//"1h",
	}
	for _, u := range newConf.Users {
		newU := types.KubeAuthUser{
			Name: u.Name,
			User: types.KubeAuthInfo{
				Exec: &types.KubeExecConfig{
					Command:            commandName,
					Args:               args,
					APIVersion:         getApiVersion(getCredentialOpts.apiVersion),
					InstallHint:        "",
					ProvideClusterInfo: false,
					InteractiveMode:    types.NeverExecInteractiveMode,
				},
			},
		}
		users = append(users, newU)
	}
	newConf.Users = users
	return newConf
}

func setupGetKubeconfigCmd(rootCmd *cobra.Command) {
	rootCmd.AddCommand(getKubeconfigCmd)
	getKubeconfigCmd.Flags().StringVarP(&getCredentialOpts.clusterId, "cluster-id", "c", "", "The cluster id to use")
	err := getKubeconfigCmd.MarkFlagRequired("cluster-id")
	common.ExitIfError(err)

	//getKubeconfigCmd.Flags().DurationVar(&getCredentialOpts.temporaryDuration, "expiration", time.Hour, "The credential expiration")
	getKubeconfigCmd.Flags().BoolVar(&getCredentialOpts.privateIpAddress, "private-address", getCredentialOpts.privateIpAddress, "Use private ip as api-server address")
	//getKubeconfigCmd.Flags().StringVar(&getCredentialOpts.apiVersion, "api-version", "v1beta1", "v1 or v1beta1")
	//getKubeconfigCmd.Flags().StringVar(&getCredentialOpts.cacheDir, "credential-cache-dir", defaultCacheDir, "Directory to cache credential")
	//getcredentialCmd.Flags().BoolVar(&getCredentialOpts.disableCache, "disable-credential-cache", false, "disable credential cache")
}