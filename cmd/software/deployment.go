package software

import (
	"fmt"
	"io"

	"github.com/astronomer/astro-cli/pkg/input"
	"github.com/astronomer/astro-cli/software/deployment"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	celeryExecutorArg     = "celery"
	localExecutorArg      = "local"
	kubernetesExecutorArg = "kubernetes"
	k8sExecutorArg        = "k8s"

	cliDeploymentHardDeletePrompt = "\nWarning: This action permanently deletes all data associated with this Deployment, including the database. You will not be able to recover it. Proceed with hard delete?"
)

var (
	allDeployments              bool
	cancel                      bool
	hardDelete                  bool
	executor                    string
	airflowVersion              string
	deploymentCreateLabel       string
	deploymentUpdateLabel       string
	deploymentUpdateDescription string
	// have to use two different executor flags for create and update commands otherwise both commands override this value
	executorUpdate          string
	deploymentID            string
	desiredAirflowVersion   string
	cloudRole               string
	releaseName             string
	nfsLocation             string
	dagDeploymentType       string
	triggererReplicas       int
	gitRevision             string
	gitRepoURL              string
	gitBranchName           string
	gitDAGDir               string
	gitSyncInterval         int
	sshKey                  string
	knowHosts               string
	deploymentCreateExample = `
# Create new deployment with Celery executor (default: celery without params).
  $ astro deployment create --label=new-deployment-name --executor=celery

# Create new deployment with Local executor.
  $ astro deployment create --label=new-deployment-name-local --executor=local

# Create new deployment with Kubernetes executor.
  $ astro deployment create --label=new-deployment-name-k8s --executor=k8s

# Create new deployment with Kubernetes executor.
  $ astro deployment create --label=my-new-deployment --executor=k8s --airflow-version=1.10.10
`
	createExampleDagDeployment = `
# Create new deployment with Kubernetes executor and dag deployment type volume and nfs location.
  $ astro deployment create --label=my-new-deployment --executor=k8s --airflow-version=2.0.0 --dag-deployment-type=volume --nfs-location=test:/test
`
	deploymentAirflowUpgradeExample = `
  $ astro deployment airflow upgrade --deployment-id=<deployment-id> --desired-airflow-version=<desired-airflow-version>

# Abort the initial airflow upgrade step:
  $ astro deployment airflow upgrade --cancel --deployment-id=<deployment-id>
`
)

func newDeploymentRootCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment",
		Aliases: []string{"de"},
		Short:   "Manage Astronomer Deployments",
		Long:    "Deployments are individual Airflow clusters running on an installation of the Astronomer platform.",
	}
	cmd.PersistentFlags().StringVar(&workspaceID, "workspace-id", "", "ID of the workspace in which you want to manage deployments, you can leave it empty if you want to use your current context's workspace ID")
	cmd.AddCommand(
		newDeploymentCreateCmd(out),
		newDeploymentListCmd(out),
		newDeploymentUpdateCmd(out),
		newDeploymentDeleteCmd(out),
		newLogsCmd(out),
		newDeploymentSaRootCmd(out),
		newDeploymentUserRootCmd(out),
		newDeploymentAirflowRootCmd(out),
	)
	return cmd
}

func newDeploymentCreateCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"cr"},
		Short:   "Create a new Astronomer Deployment",
		Long:    "Create a new Astronomer Deployment",
		Example: deploymentCreateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentCreate(cmd, out)
		},
	}

	var nfsMountDAGDeploymentEnabled, triggererEnabled, gitSyncDAGDeploymentEnabled bool
	appConfig, err := houstonClient.GetAppConfig()
	if err != nil {
		initDebugLogs = append(initDebugLogs, fmt.Sprintf("Error checking feature flag: %s", err.Error()))
	} else {
		nfsMountDAGDeploymentEnabled = appConfig.Flags.NfsMountDagDeployment
		triggererEnabled = appConfig.Flags.TriggererEnabled
		gitSyncDAGDeploymentEnabled = appConfig.Flags.GitSyncEnabled
	}

	// let's hide under feature flag
	if nfsMountDAGDeploymentEnabled || gitSyncDAGDeploymentEnabled {
		cmd.Flags().StringVarP(&dagDeploymentType, "dag-deployment-type", "t", "", "DAG Deployment mechanism: image, volume, git_sync")
	}

	if nfsMountDAGDeploymentEnabled {
		cmd.Example += createExampleDagDeployment
		cmd.Flags().StringVarP(&nfsLocation, "nfs-location", "n", "", "NFS Volume Mount, specified as: <IP>:/<path>. Input is automatically prepended with 'nfs://' - do not include.")
	}

	if gitSyncDAGDeploymentEnabled {
		addGitSyncDeploymentFlags(cmd)
	}

	if triggererEnabled {
		cmd.Flags().IntVarP(&triggererReplicas, "triggerer-replicas", "", 0, "Number of replicas to use for triggerer airflow component, valid 0-2")
	}

	cmd.Flags().StringVarP(&deploymentCreateLabel, "label", "l", "", "Label of your deployment")
	cmd.Flags().StringVarP(&executor, "executor", "e", celeryExecutorArg, "The executor used in your Airflow deployment, one of: local, celery, or kubernetes")
	cmd.Flags().StringVarP(&airflowVersion, "airflow-version", "a", "", "Add desired airflow version parameter: e.g: 1.10.5 or 1.10.7")
	cmd.Flags().StringVarP(&releaseName, "release-name", "r", "", "Set custom release-name if possible")
	cmd.Flags().StringVarP(&cloudRole, "cloud-role", "c", "", "Set cloud role to annotate service accounts in deployment")
	_ = cmd.MarkFlagRequired("label")
	return cmd
}

func newDeploymentDeleteCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [deployment ID]",
		Aliases: []string{"de"},
		Short:   "Delete an airflow deployment",
		Long:    "Delete an airflow deployment",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentDelete(cmd, args, out)
		},
	}
	if deployment.CheckHardDeleteDeployment(houstonClient) {
		cmd.Flags().BoolVar(&hardDelete, "hard", false, "Deletes all infrastructure and records for this Deployment")
	}
	return cmd
}

func newDeploymentListCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List airflow deployments",
		Long:    "List airflow deployments",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentList(cmd, out)
		},
	}
	cmd.Flags().BoolVarP(&allDeployments, "all", "a", false, "Show deployments across all workspaces")
	return cmd
}

func newDeploymentUpdateCmd(out io.Writer) *cobra.Command {
	example := `
# update executor for given deployment
$ astro deployment update [deployment ID] --executor=celery`
	updateExampleDagDeployment := `

# update dag deployment strategy
$ astro deployment update [deployment ID] --dag-deployment-type=volume --nfs-location=test:/test`
	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"up"},
		Short:   "Update airflow deployments",
		Long:    "Update airflow deployments",
		Example: example,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentUpdate(cmd, args, dagDeploymentType, nfsLocation, out)
		},
	}

	var nfsMountDAGDeploymentEnabled, triggererEnabled, gitSyncDAGDeploymentEnabled bool
	appConfig, err := houstonClient.GetAppConfig()
	if err != nil {
		initDebugLogs = append(initDebugLogs, fmt.Sprintf("Error checking feature flag: %s", err.Error()))
	} else {
		nfsMountDAGDeploymentEnabled = appConfig.Flags.NfsMountDagDeployment
		triggererEnabled = appConfig.Flags.TriggererEnabled
		gitSyncDAGDeploymentEnabled = appConfig.Flags.GitSyncEnabled
	}

	cmd.Flags().StringVarP(&executorUpdate, "executor", "e", "", "Add executor parameter: local, celery, or kubernetes")

	// let's hide under feature flag
	if nfsMountDAGDeploymentEnabled || gitSyncDAGDeploymentEnabled {
		cmd.Flags().StringVarP(&dagDeploymentType, "dag-deployment-type", "t", "", "DAG Deployment mechanism: image, volume, git_sync")
	}

	if nfsMountDAGDeploymentEnabled {
		cmd.Example += updateExampleDagDeployment
		cmd.Flags().StringVarP(&nfsLocation, "nfs-location", "n", "", "NFS Volume Mount, specified as: <IP>:/<path>. Input is automatically prepended with 'nfs://' - do not include.")
	}

	if triggererEnabled {
		cmd.Flags().IntVarP(&triggererReplicas, "triggerer-replicas", "", 0, "Number of replicas to use for triggerer airflow component, valid 0-2")
	}

	//noline:dupl
	if gitSyncDAGDeploymentEnabled {
		addGitSyncDeploymentFlags(cmd)
	}

	cmd.Flags().StringVarP(&deploymentUpdateDescription, "description", "d", "", "Set description to update in deployment")
	cmd.Flags().StringVarP(&deploymentUpdateLabel, "label", "l", "", "Set label to update in deployment")
	cmd.Flags().StringVarP(&cloudRole, "cloud-role", "c", "", "Set cloud role to annotate service accounts in deployment")
	return cmd
}

func addGitSyncDeploymentFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&gitRevision, "git-revision", "v", "", "Git revision (tag or hash) to check out")
	cmd.Flags().StringVarP(&gitRepoURL, "git-repository-url", "u", "", "The repository URL of the git repo")
	cmd.Flags().StringVarP(&gitBranchName, "git-branch-name", "b", "", "The Branch name of the git repo we will be syncing from")
	cmd.Flags().StringVarP(&gitDAGDir, "dag-directory-path", "p", "", "The directory where dags are stored in repo")
	cmd.Flags().IntVarP(&gitSyncInterval, "sync-interval", "s", 1, "The interval in seconds in which git-sync will be polling git for updates")
	cmd.Flags().StringVarP(&sshKey, "ssh-key", "", "", "Path to the ssh public key file to use to clone your git repo")
	cmd.Flags().StringVarP(&knowHosts, "known-hosts", "", "", "Path to the known hosts file to use to clone your git repo")
}

func newDeploymentAirflowRootCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "airflow",
		Aliases: []string{"ai"},
		Short:   "Manage airflow deployments",
		Long:    "Manage airflow deployments",
	}
	cmd.AddCommand(
		newDeploymentAirflowUpgradeCmd(out),
	)
	return cmd
}

func newDeploymentAirflowUpgradeCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"up"},
		Short:   "Upgrade Airflow version",
		Long:    "Upgrade Airflow version",
		Example: deploymentAirflowUpgradeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentAirflowUpgrade(cmd, out)
		},
	}
	cmd.Flags().StringVarP(&deploymentID, "deployment-id", "d", "", "ID of the deployment to upgrade")
	cmd.Flags().StringVarP(&desiredAirflowVersion, "desired-airflow-version", "v", "", "Desired Airflow version to upgrade to")
	cmd.Flags().BoolVarP(&cancel, "cancel", "c", false, "Abort the initial airflow upgrade step")
	err := cmd.MarkFlagRequired("deployment-id")
	if err != nil {
		fmt.Println("error adding deployment-id flag: ", err.Error())
	}
	return cmd
}

func deploymentCreate(cmd *cobra.Command, out io.Writer) error {
	ws, err := coalesceWorkspace()
	if err != nil {
		return fmt.Errorf("failed to find a valid workspace: %w", err)
	}

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	executorType, err := validateExecutorArg(executor)
	if err != nil {
		return err
	}

	var nfsMountDAGDeploymentEnabled, gitSyncDAGDeploymentEnabled bool
	appConfig, err := houstonClient.GetAppConfig()
	if err != nil {
		logrus.Debugln("Error checking feature flag", err)
	} else {
		nfsMountDAGDeploymentEnabled = appConfig.Flags.NfsMountDagDeployment
		gitSyncDAGDeploymentEnabled = appConfig.Flags.GitSyncEnabled
	}

	// we should validate only in case when this feature has been enabled
	if nfsMountDAGDeploymentEnabled || gitSyncDAGDeploymentEnabled {
		err = validateDagDeploymentArgs(dagDeploymentType, nfsLocation, gitRepoURL, false)
		if err != nil {
			return err
		}
	}

	return deployment.Create(deploymentCreateLabel, ws, releaseName, cloudRole, executorType, airflowVersion, dagDeploymentType, nfsLocation, gitRepoURL, gitRevision, gitBranchName, gitDAGDir, sshKey, knowHosts, gitSyncInterval, triggererReplicas, houstonClient, out)
}

func deploymentDelete(cmd *cobra.Command, args []string, out io.Writer) error {
	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true
	if hardDelete {
		i, _ := input.Confirm(cliDeploymentHardDeletePrompt)

		if !i {
			fmt.Println("Exit: This command was not executed and your Deployment was not hard deleted.\n If you want to delete your Deployment but not permanently, try\n $ astro deployment delete without the --hard flag.")
			return nil
		}
	}
	return deployment.Delete(args[0], hardDelete, houstonClient, out)
}

func deploymentList(cmd *cobra.Command, out io.Writer) error {
	ws, err := coalesceWorkspace()
	if err != nil {
		return fmt.Errorf("failed to find a valid workspace: %w", err)
	}

	// Don't validate workspace if viewing all deployments
	if allDeployments {
		ws = ""
	}

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	return deployment.List(ws, allDeployments, houstonClient, out)
}

func deploymentUpdate(cmd *cobra.Command, args []string, dagDeploymentType, nfsLocation string, out io.Writer) error {
	argsMap := map[string]string{}
	if deploymentUpdateDescription != "" {
		argsMap["description"] = deploymentUpdateDescription
	}
	if deploymentUpdateLabel != "" {
		argsMap["label"] = deploymentUpdateLabel
	}

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	var nfsMountDAGDeploymentEnabled, gitSyncDAGDeploymentEnabled bool
	appConfig, err := houstonClient.GetAppConfig()
	if err != nil {
		logrus.Debugln("Error checking feature flag", err)
	} else {
		nfsMountDAGDeploymentEnabled = appConfig.Flags.NfsMountDagDeployment
		gitSyncDAGDeploymentEnabled = appConfig.Flags.GitSyncEnabled
	}

	// we should validate only in case when this feature has been enabled
	if nfsMountDAGDeploymentEnabled || gitSyncDAGDeploymentEnabled {
		err = validateDagDeploymentArgs(dagDeploymentType, nfsLocation, gitRepoURL, true)
		if err != nil {
			return err
		}
	}

	var executorType string
	if executorUpdate != "" {
		executorType, err = validateExecutorArg(executorUpdate)
		if err != nil {
			return nil
		}
	}

	return deployment.Update(args[0], cloudRole, argsMap, dagDeploymentType, nfsLocation, gitRepoURL, gitRevision, gitBranchName, gitDAGDir, sshKey, knowHosts, executorType, gitSyncInterval, triggererReplicas, houstonClient, out)
}

func deploymentAirflowUpgrade(cmd *cobra.Command, out io.Writer) error {
	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true
	if cancel {
		return deployment.AirflowUpgradeCancel(deploymentID, houstonClient, out)
	}
	return deployment.AirflowUpgrade(deploymentID, desiredAirflowVersion, houstonClient, out)
}