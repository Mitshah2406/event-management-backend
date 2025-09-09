package response

import "github.com/gin-gonic/gin"

func RespondJSON(c *gin.Context, status string, code int, message string, data interface{}, errors interface{}) {
	c.JSON(code, StandardApiResponse{
		Status:     status,
		StatusCode: code,
		Message:    message,
		Data:       data,
		Errors:     errors,
	})
}
