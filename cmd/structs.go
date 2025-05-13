package cmd

type DeployOptions struct {
	Stage string
}

type InitOptions struct {
	ProjectName string
	Stage       string
	S3Bucket    string
}

type DeploymentConfig struct {
	FunctionName string
	S3Bucket     string
	S3Key        string
	Timeout      int
	Memory       int
	Stage        string
}
