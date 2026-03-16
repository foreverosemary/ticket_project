package utils

import (
	"os"
	"ticket/utils/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type MyClaims struct {
	UserID int64 `json:"userId"`
	RoleID int   `json:"roleId"`
	jwt.RegisteredClaims
}

var secretKey []byte

func init() {
	key, err := os.ReadFile("./config/secret.txt")
	if err != nil {
		panic("加载密钥失败: " + err.Error())
	}
	secretKey = key
}

func GenerateToken(userId int64, roleID int) (string, error) {
	claims := MyClaims{
		UserID: userId,
		RoleID: roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

func ParseToken(tokenStr string) (*MyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Token
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			response.JsonErr(c, 401, "未登录或未携带 Token")
			c.Abort()
			return
		}

		// 解析 Token
		claims, err := ParseToken(tokenStr)

		if err != nil {
			response.JsonErr(c, 401, "Token 无效或已过期")
			c.Abort()
			return
		}

		// 滑动过期
		expireTime := claims.ExpiresAt.Time
		now := time.Now()

		if expireTime.Sub(now) < 30*time.Minute {
			newToken, _ := GenerateToken(claims.UserID, claims.RoleID)
			c.Header("New_Token", newToken)
		}

		// 存取用户信息
		c.Set("userID", claims.UserID)
		c.Set("roleID", claims.RoleID)
		c.Next()
	}
}

func Permissed(c *gin.Context) {
	roleID := c.GetInt("roleID")
	if roleID != 1 {
		response.JsonErr(c, 403, "无权限")
		c.Abort()
	}
}

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		JWTAuth()(c)

		if c.IsAborted() {
			return
		}

		Permissed(c)
	}
}
