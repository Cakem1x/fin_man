package prompt

import (
	"errors"

	"github.com/charmbracelet/huh"
)

// ReadPassword prompts the user for a single password with masked input.
func ReadPassword(title string) (string, error) {
	var password string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				EchoMode(huh.EchoModePassword).
				Value(&password),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return password, nil
}

// ReadNewPassword prompts the user for a new password and confirmation with inline validation.
func ReadNewPassword(title, confirmTitle string) (string, error) {
	var password string
	var confirmPassword string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				EchoMode(huh.EchoModePassword).
				Value(&password),
			huh.NewInput().
				Title(confirmTitle).
				EchoMode(huh.EchoModePassword).
				Value(&confirmPassword).
				Validate(func(str string) error {
					if str != password {
						return errors.New("passwords do not match")
					}
					return nil
				}),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return password, nil
}
