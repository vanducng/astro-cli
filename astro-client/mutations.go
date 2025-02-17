package astro

var (
	CreateImage = `
	mutation CreateImage($imageCreateInput: CreateImageInput!) {
		createImage(input: $imageCreateInput) {
			id
			deploymentId
			tag
		}
	}
	`

	DeployImage = `
	mutation DeployImage($imageDeployInput: DeployImageInput!) {
		deployImage(input: $imageDeployInput) {
			id
			tag
			repository
		}
	}
	`

	DeleteDeployment = `
	mutation deleteDeployment(
		$input: DeleteDeploymentInput!
	  ) {
		deleteDeployment(
			input: $input
		) {
		  id
		}
	}
	`

	CreateDeployment = `
	mutation createDeployment(
		$input: CreateDeploymentInput!
	  ) {
		createDeployment (
		  input: $input
		){
			id
			label
			releaseName
			dagDeployEnabled
			schedulerSize
			type
			isHighAvailability
			apiKeyOnlyDeployments
			cluster {
				id
				name
				region
			}
			runtimeRelease {
				version
				airflowVersion
			}
			workerQueues {
				id
				name
				astroMachine
				isDefault
				workerConcurrency
				minWorkerCount
				maxWorkerCount
				nodePoolId
				podCpu
				podRam
			}
			deploymentSpec {
				image {
					tag
				}
				webserver {
					url
				}
			}
		}
	}
	`

	UpdateDeployment = `
	mutation updateDeployment(
		$input: UpdateDeploymentInput!
	  ) {
		updateDeployment(
			input: $input
		) {
			id
			label
			releaseName
			dagDeployEnabled
			schedulerSize
			type
			isHighAvailability
			apiKeyOnlyDeployments
			cluster {
				id
				name
				region
			}
			runtimeRelease {
				version
				airflowVersion
			}
			workerQueues {
				id
				name
				astroMachine
				isDefault
				workerConcurrency
				minWorkerCount
				maxWorkerCount
				nodePoolId
				podCpu
				podRam
			}
			deploymentSpec {
				image {
					tag
				}
			}
	  	}
	}
	`
	CreateDeploymentVariables = `
	mutation updateDeploymentVariables(
	  $input: EnvironmentVariablesInput!
	) {
		updateDeploymentVariables(
		input: $input
	  ) {
		key
		value
		isSecret
		updatedAt
	  }
	}
  `
	DagDeploymentInitiate = `
	mutation initiateDagDeployment($input: InitiateDagDeploymentInput!) {
		initiateDagDeployment(input: $input) {
			id
			dagUrl
		}
	}
  `
	ReportDagDeploymentStatus = `
	mutation reportDagDeploymentStatus($input: ReportDagDeploymentStatusInput!) {
		reportDagDeploymentStatus(input: $input) {
			id
			runtimeId
			action
			versionId
			status
			message
			createdAt
			initiatorId
			initiatorType
		}
	}
  `
	UpdateDeploymentAlerts = `
	mutation updateDeploymentAlerts($input: UpdateDeploymentAlertsInput!) {
		updateDeploymentAlerts(input: $input) {
			alertEmails
		}
	}
  `
)
