package restic

func (ResticBackend) SupportsRun() bool {
	return true
}

func (ResticBackend) Run(path string, args []string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	invocParams, err := generateResticDirectInvocation(config)
	if err != nil {
		return err
	}

	invocParams.Args = append(invocParams.Args, args...)

	return invocParams.Execute()

}
