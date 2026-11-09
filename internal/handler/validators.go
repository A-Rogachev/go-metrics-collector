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

func getErrorResponse(errors *validator.ValidationErrors) []ValidationErrorResponse {
	errorsResponse := make([]ValidationErrorResponse, 0, len(*errors))
	for _, validationError := range *errors {
		errorsResponse = append(errorsResponse, ValidationErrorResponse{
			Field:   validationError.Field(),
			Tag:     validationError.Tag(),
			Value:   validationError.Value(),
			Message: validationError.Error(),
		})
	}
	return errorsResponse
}
