package cmd

type DeployOptions struct {
	Stage string
}

type UpdateOptions struct {
	Stage string
}

type UndeployOptions struct {
	Stage string
	Force bool
}

type InitOptions struct {
	ProjectName string
	Stage       string
	S3Bucket    string
	Timeout     int
	Memory      int
}

type DeploymentConfig struct {
	FunctionName string
	S3Bucket     string
	S3Key        string
	Timeout      int
	Memory       int
	Stage        string
}
