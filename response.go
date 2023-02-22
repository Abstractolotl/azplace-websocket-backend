package main

type Response struct {
	Error   bool                    `json:"error"`
	Message *string                 `json:"message"`
	Method  *string                 `json:"method"`
	Data    *map[string]interface{} `json:"data"`
}

func NewResponse(error bool, message string, method string, data *map[string]interface{}) *Response {
	response := new(Response)

	response.Error = error

	if len(message) > 0 {
		response.Message = &message
	}

	if len(method) > 0 {
		response.Method = &method
	}

	response.Data = data

	return response
}
