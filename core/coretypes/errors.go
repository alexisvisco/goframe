package coretypes

import (
	"encoding/json"

	z "github.com/Oudwins/zog"
)

type ValidationError struct {
	Errors map[string][]string
}

func (v *ValidationError) Error() string {
	marshal, err := json.Marshal(v.Errors)
	if err != nil {
		return "validation error"
	}

	return string(marshal)
}

func ValidationErrorFromZog(errs z.ZogIssueMap) error {
	if len(errs) == 0 {
		return nil
	}
	codeErrs := make(map[string][]string, len(errs))
	for k, v := range errs {
		if k == "$first" {
			continue
		}
		e := make([]string, len(v))
		for i, err := range v {
			e[i] = err.Code
		}

		codeErrs[k] = e
	}

	return &ValidationError{
		Errors: codeErrs,
	}
}
