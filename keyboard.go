package ssh

import "fmt"

type keyboardInteractive map[string]string

func (cr keyboardInteractive) Challenge(user, instruction string, questions []string, echos []bool) ([]string, error) {
	var answers []string
	for _, question := range questions {
		answer, ok := cr[question]
		if !ok {
			return nil, fmt.Errorf("question[%s] not answer", question)
		}
		answers = append(answers, answer)
	}
	return answers, nil
}
