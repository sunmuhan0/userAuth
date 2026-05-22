package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ttuser/proc/sp"
)

// AuthHandler HTTP接口处理器
// 通过 sp.Get().AuthManager 调用 gRPC 服务
type AuthHandler struct{}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LogoutRequest 注销请求
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest 续签请求
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UpdateUserInfoRequest 更新用户信息请求
type UpdateUserInfoRequest struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// Login 登录接口
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := sp.Get().AuthManager.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"access_token":  resp.Token,
			"refresh_token": resp.RefreshToken,
			"expires_at":    resp.ExpiresAt,
			"user": gin.H{
				"id":       resp.User.Id,
				"username": resp.User.Username,
				"nickname": resp.User.Nickname,
				"email":    resp.User.Email,
				"avatar":   resp.User.Avatar,
			},
		},
	})
}

// Logout 注销接口
func (h *AuthHandler) Logout(c *gin.Context) {
	// access_token 从 context 中获取（鉴权中间件已写入）
	accessToken, _ := c.Get("token")

	// refresh_token 从请求body中获取
	var req LogoutRequest
	_ = c.ShouldBindJSON(&req)

	accessTokenStr := ""
	if accessToken != nil {
		accessTokenStr = accessToken.(string)
	}

	_, err := sp.Get().AuthManager.Logout(c.Request.Context(), accessTokenStr, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "logout failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "logout success",
	})
}

// Refresh 续签接口（公开路由，无需Bearer鉴权）
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := sp.Get().AuthManager.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "refresh failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"access_token":  resp.AccessToken,
			"refresh_token": resp.RefreshToken,
			"expires_at":    resp.ExpiresAt,
		},
	})
}

// GetUserInfo 获取当前用户信息
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	resp, err := sp.Get().AuthManager.GetUserInfo(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":         resp.User.Id,
			"username":   resp.User.Username,
			"nickname":   resp.User.Nickname,
			"email":      resp.User.Email,
			"avatar":     resp.User.Avatar,
			"created_at": resp.User.CreatedAt,
			"updated_at": resp.User.UpdatedAt,
		},
	})
}

// UpdateUserInfo 更新用户信息
func (h *AuthHandler) UpdateUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	var req UpdateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := sp.Get().AuthManager.UpdateUserInfo(c.Request.Context(), userID.(string), req.Nickname, req.Email, req.Avatar)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":         resp.User.Id,
			"username":   resp.User.Username,
			"nickname":   resp.User.Nickname,
			"email":      resp.User.Email,
			"avatar":     resp.User.Avatar,
			"created_at": resp.User.CreatedAt,
			"updated_at": resp.User.UpdatedAt,
		},
	})
}
