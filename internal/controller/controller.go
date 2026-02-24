package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"rental-architecture-back/internal/db"
	"rental-architecture-back/internal/services"
)

type Deps struct {
	DB        *db.DB
	Auth      *services.AuthService
	Audit     *services.AuditService
	Analytics *services.AnalyticsService
	Sim       *services.Engine
}

type Controller struct {
	deps Deps
}

func New(d Deps) *Controller {
	return &Controller{deps: d}
}

func (c *Controller) Mount(r *gin.Engine) {
	r.GET("/openapi.yaml", func(ctx *gin.Context) { ctx.File("openapi.yaml") })
	r.GET("/docs", func(ctx *gin.Context) { ctx.File("docs/swagger.html") })

	authG := r.Group("/auth")
	authG.POST("/register", c.register)
	authG.POST("/login", c.login)

	authorized := r.Group("")
	authorized.Use(c.authMiddleware())

	cells := authorized.Group("/cells")
	cells.POST("/", c.createCell)
	cells.GET("/nearby", c.cellsNearby)
	cells.GET("/:h3", c.getCellByH3)

	sim := authorized.Group("/simulation")
	sim.POST("/run", c.runSimulation)
	sim.GET("/runs/:id", c.getSimulationRun)

	admin := authorized.Group("/audit")
	admin.Use(c.requireRole("admin"))
	admin.GET("/", c.listAudit)

	r.POST("/analytics/recompute", c.recomputeAnalytics)
	r.GET("/analytics", c.listAnalytics)
}

func (c *Controller) authMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h := ctx.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tok := strings.TrimPrefix(h, "Bearer ")
		claims, err := c.deps.Auth.ParseToken(tok)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		ctx.Set("claims", claims)
		ctx.Next()
	}
}

func (c *Controller) requireRole(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		v, _ := ctx.Get("claims")
		cl, _ := v.(*services.Claims)
		if cl == nil || cl.Role != role {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx.Next()
	}
}

func getClaimsFrom(ctx *gin.Context) *services.Claims {
	v, _ := ctx.Get("claims")
	if v == nil {
		return nil
	}
	c, _ := v.(*services.Claims)
	return c
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (c *Controller) register(ctx *gin.Context) {
	var req registerReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
		return
	}
	if req.Email == "" || req.Password == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "email/password required"})
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	if req.Role != "admin" && req.Role != "user" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or user"})
		return
	}

	hash, err := c.deps.Auth.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "hash error"})
		return
	}

	_, err = c.deps.DB.SQL.Exec(`INSERT INTO users(email, password_hash, role, created_at) VALUES($1,$2,$3,$4)`, req.Email, hash, req.Role, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user exists or db error"})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"ok": true})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c *Controller) login(ctx *gin.Context) {
	var req loginReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
		return
	}
	if req.Email == "" || req.Password == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "email/password required"})
		return
	}

	var id int64
	var hash, role string
	err := c.deps.DB.SQL.QueryRow(`SELECT id, password_hash, role FROM users WHERE email=$1`, req.Email).Scan(&id, &hash, &role)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !c.deps.Auth.CheckPassword(hash, req.Password) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := c.deps.Auth.IssueToken(id, role)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}
	_ = c.deps.Audit.Log(&id, "auth.login", "user:"+req.Email)
	ctx.JSON(http.StatusOK, gin.H{"token": token})
}

type createCellReq struct {
	Name       string  `json:"name"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Resolution int     `json:"resolution"`
}

func (c *Controller) createCell(ctx *gin.Context) {
	claims := getClaimsFrom(ctx)
	if claims == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req createCellReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
		return
	}
	if req.Name == "" {
		req.Name = "cell"
	}
	if req.Resolution <= 0 {
		req.Resolution = 9
	}
	h3i := services.LatLngToH3(req.Lat, req.Lng, req.Resolution)

	var id int64
	err := c.deps.DB.SQL.QueryRow(`INSERT INTO cells(name, lat, lng, h3_index, resolution, created_by, created_at) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, req.Name, req.Lat, req.Lng, h3i, req.Resolution, claims.UserID, time.Now().UTC().Format(time.RFC3339)).Scan(&id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	uid := int64(claims.UserID)
	_ = c.deps.Audit.Log(&uid, "cells.create", "h3:"+h3i)
	ctx.JSON(http.StatusCreated, gin.H{"id": id, "h3": h3i})
}

func (c *Controller) getCellByH3(ctx *gin.Context) {
	h3i := ctx.Param("h3")
	if strings.TrimSpace(h3i) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "h3 required"})
		return
	}
	claims := getClaimsFrom(ctx)
	if claims == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	q := `SELECT id, name, lat, lng, h3_index, resolution, created_by, created_at FROM cells WHERE h3_index=$1`
	args := []any{h3i}
	if claims.Role == "user" {
		q += " AND created_by=$2"
		args = append(args, claims.UserID)
	}
	rows, err := c.deps.DB.SQL.Query(q, args...)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var id, res, createdBy int64
		var name, h3s, createdAt string
		var lat, lng float64
		if err := rows.Scan(&id, &name, &lat, &lng, &h3s, &res, &createdBy, &createdAt); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, map[string]any{"id": id, "name": name, "lat": lat, "lng": lng, "h3": h3s, "resolution": res, "createdBy": createdBy, "createdAt": createdAt})
	}
	ctx.JSON(http.StatusOK, gin.H{"items": out})
}

func (c *Controller) cellsNearby(ctx *gin.Context) {
	claims := getClaimsFrom(ctx)
	if claims == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	center := ctx.Query("h3")
	kStr := ctx.Query("k")
	if center == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "h3 query param required"})
		return
	}
	k := 1
	if kStr != "" {
		if v, err := strconv.Atoi(kStr); err == nil {
			k = v
		}
	}
	if k < 0 {
		k = 0
	}
	if k > 5 {
		k = 5
	}
	ring, err := services.KRing(center, k)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid h3"})
		return
	}

	placeParts := make([]string, 0, len(ring))
	args := make([]any, 0, len(ring)+1)
	for i, x := range ring {
		placeParts = append(placeParts, "$"+strconv.Itoa(i+1))
		args = append(args, x)
	}
	place := strings.Join(placeParts, ",")
	q := `SELECT id, name, lat, lng, h3_index, resolution, created_by, created_at FROM cells WHERE h3_index IN (` + place + `)`
	if claims.Role == "user" {
		q += " AND created_by=$" + strconv.Itoa(len(args)+1)
		args = append(args, claims.UserID)
	}
	q += " ORDER BY id DESC LIMIT 200"
	rows, err := c.deps.DB.SQL.Query(q, args...)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var id, res, createdBy int64
		var name, h3s, createdAt string
		var lat, lng float64
		if err := rows.Scan(&id, &name, &lat, &lng, &h3s, &res, &createdBy, &createdAt); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, map[string]any{"id": id, "name": name, "lat": lat, "lng": lng, "h3": h3s, "resolution": res, "createdBy": createdBy, "createdAt": createdAt})
	}
	ctx.JSON(http.StatusOK, gin.H{"center": center, "k": k, "items": out})
}

func (c *Controller) runSimulation(ctx *gin.Context) {
	claims := getClaimsFrom(ctx)
	if claims == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var rq struct {
		Ticks int `json:"ticks"`
	}
	_ = ctx.BindJSON(&rq)
	id, err := c.deps.Sim.CreateRun(int64(claims.UserID), rq.Ticks)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "create run failed"})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"id": id})
}

func (c *Controller) getSimulationRun(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	data, err := c.deps.Sim.GetRun(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "not found"})
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (c *Controller) recomputeAnalytics(ctx *gin.Context) {
	resStr := ctx.Query("res")
	targetRes := 7
	if resStr != "" {
		if v, err := strconv.Atoi(resStr); err == nil {
			targetRes = v
		}
	}
	if err := c.deps.Analytics.RecomputeCounts(targetRes); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "recompute failed"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (c *Controller) listAnalytics(ctx *gin.Context) {
	resStr := ctx.Query("res")
	limitStr := ctx.Query("limit")
	targetRes := 7
	if resStr != "" {
		if v, err := strconv.Atoi(resStr); err == nil {
			targetRes = v
		}
	}
	limit := 200
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}
	rows, err := c.deps.Analytics.ListCounts(targetRes, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": rows})
}

func (c *Controller) listAudit(ctx *gin.Context) {
	limit := 50
	if l := ctx.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	rows, err := c.deps.Audit.List(limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": rows})
}
