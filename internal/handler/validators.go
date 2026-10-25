package handler

import (
	"github.com/go-playground/validator/v10"
)

type MetricValidator struct {
	Validator *validator.Validate
}

type ValidationErrorResponse struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   any    `json:"value"`
	Message string `json:"message"`
}

func (mv *MetricValidator) Validate(i any) error {
	if err := mv.Validator.Struct(i); err != nil {
		return err
	}
	return nil
}
