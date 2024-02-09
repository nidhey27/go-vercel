package main

import (
	"os/exec"
)

func buildProject(projectPath string) error {

	command := exec.Command("/bin/bash", "-c", "npm i && npm run build")
	command.Dir = projectPath
	cmdErr := command.Run()

	return cmdErr
}
