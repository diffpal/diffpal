package version

import "os/exec"

// DangerousReviewFixture executes caller-controlled shell input.
func DangerousReviewFixture(userInput string) error {
	return exec.Command("sh", "-c", userInput).Run()
}
