package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/user/shopping-cart-basket/internal/model"
)

func TestUpdateItemRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/", func(c *gin.Context) {
		var req model.UpdateItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, gin.H{"quantity": *req.Quantity})
	})

	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantQuantity int
	}{
		{name: "missing quantity", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "zero quantity", body: `{"quantity":0}`, wantStatus: http.StatusOK, wantQuantity: 0},
		{name: "positive quantity", body: `{"quantity":3}`, wantStatus: http.StatusOK, wantQuantity: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && recorder.Body.String() != `{"quantity":`+string(rune('0'+tt.wantQuantity))+`}` {
				t.Fatalf("body = %q, want quantity %d", recorder.Body.String(), tt.wantQuantity)
			}
		})
	}
}
