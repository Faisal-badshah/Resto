package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang-jwt/jwt/v4"
)

var sanitizeRe = regexp.MustCompile(`[^a-zA-Z0-9\-_\. ]+`)

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = sanitizeRe.ReplaceAllString(name, "_")
	if len(name) > 120 {
		ext := filepath.Ext(name)
		name = name[:110] + ext
	}
	return name
}

func deriveExt(src string) string {
	ext := filepath.Ext(src)
	ext = strings.ToLower(ext)
	if ext == "" || len(ext) > 6 {
		if strings.Contains(strings.ToLower(src), "jpeg") {
			return ".jpeg"
		}
		return ".jpg"
	}
	return ext
}

func createZipEntryName(kind string, category string, itemName string, src string, idx int) string {
	ext := deriveExt(src)
	base := "item"
	if itemName != "" {
		base = sanitizeFilename(itemName)
	} else {
		base = fmt.Sprintf("asset_%d", idx)
	}
	if kind == "menu" {
		cat := "uncategorized"
		if category != "" {
			cat = sanitizeFilename(category)
		}
		return filepath.ToSlash(filepath.Join("menus", cat, base+ext))
	}
	bn := filepath.Base(src)
	bn = sanitizeFilename(bn)
	return filepath.ToSlash(filepath.Join("galleries", fmt.Sprintf("%02d_%s", idx, bn)))
}

func fetchToWriter(ctx context.Context, src string, writer io.Writer) error {
	src = strings.TrimSpace(src)
	if src == "" {
		return errors.New("empty src")
	}

	if strings.HasPrefix(src, "s3://") {
		trim := strings.TrimPrefix(src, "s3://")
		parts := strings.SplitN(trim, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid s3 url: %s", src)
		}
		bucket := parts[0]
		key := parts[1]

		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return fmt.Errorf("aws config: %w", err)
		}
		client := s3.NewFromConfig(cfg)
		out, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		})
		if err != nil {
			return fmt.Errorf("s3 get object: %w", err)
		}
		defer out.Body.Close()
		_, err = io.Copy(writer, out.Body)
		return err
	}

	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		req, err := http.NewRequestWithContext(ctx, "GET", src, nil)
		if err != nil {
			return err
		}
		client := &http.Client{Timeout: 20 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("http status %d", resp.StatusCode)
		}
		_, err = io.Copy(writer, resp.Body)
		return err
	}

	local := src
	imgRoot := os.Getenv("IMG_ROOT")
	if imgRoot == "" {
		imgRoot = "./frontend/public"
	}
	if strings.HasPrefix(src, "/") {
		local = filepath.Join(imgRoot, strings.TrimPrefix(src, "/"))
	} else {
		local = filepath.Join(imgRoot, src)
	}
	local = filepath.Clean(local)

	f, err := os.Open(local)
	if err != nil {
		return fmt.Errorf("open local: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(writer, f)
	return err
}

func (s *Server) writeZipStream(ctx context.Context, w io.Writer, images []struct {
	Src      string
	ZipName  string
	Kind     string
	Category string
	ItemName string
}, manifest []map[string]any) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, it := range images {
		fw, err := zw.Create(it.ZipName)
		if err != nil {
			fmt.Printf("zip create failed for %s: %v\n", it.ZipName, err)
			continue
		}
		if err := fetchToWriter(ctx, it.Src, fw); err != nil {
			fmt.Printf("fetch failed for %s -> %s: %v\n", it.Src, it.ZipName, err)
			continue
		}
	}

	if manifest != nil {
		mbytes, _ := json.MarshalIndent(manifest, "", "  ")
		fm, err := zw.Create("manifest.json")
		if err == nil {
			_, _ = fm.Write(mbytes)
		}
	}

	return nil
}

func (s *Server) handleExportMedia(w http.ResponseWriter, r *http.Request, claims jwt.MapClaims) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "only GET/POST allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := getIDFromPath("/api/admin/export_media/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid restaurant id", http.StatusBadRequest)
		return
	}

	role, _ := claims["role"].(string)
	if role != "owner" {
		http.Error(w, "forbidden - owner only", http.StatusForbidden)
		return
	}

	var payload struct {
		Target    string `json:"target"`
		Bucket    string `json:"bucket"`
		KeyPrefix string `json:"keyPrefix"`
		Public    bool   `json:"public"`
	}
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	ctx := r.Context()
	data, err := s.store.LoadRestaurantData(ctx, id)
	if err != nil {
		http.Error(w, "failed to load data", http.StatusInternalServerError)
		return
	}

	type imgEntry struct {
		Src      string
		ZipName  string
		Kind     string
		Category string
		ItemName string
	}
	seen := map[string]bool{}
	var images []imgEntry
	var manifest []map[string]any
	idx := 0
	for _, cat := range data.Menus {
		for _, it := range cat.Items {
			if strings.TrimSpace(it.Img) == "" {
				continue
			}
			src := it.Img
			if seen[src] {
				continue
			}
			idx++
			zn := createZipEntryName("menu", cat.Category, it.Name, src, idx)
			images = append(images, imgEntry{Src: src, ZipName: zn, Kind: "menu", Category: cat.Category, ItemName: it.Name})
			manifest = append(manifest, map[string]any{
				"zip":      zn,
				"source":   src,
				"kind":     "menu",
				"category": cat.Category,
				"itemName": it.Name,
			})
			seen[src] = true
		}
	}
	for gi, g := range data.Galleries.Images {
		if strings.TrimSpace(g) == "" {
			continue
		}
		if seen[g] {
			continue
		}
		idx++
		zn := createZipEntryName("gallery", "", "", g, gi+1)
		images = append(images, imgEntry{Src: g, ZipName: zn, Kind: "gallery"})
		manifest = append(manifest, map[string]any{
			"zip":    zn,
			"source": g,
			"kind":   "gallery",
		})
		seen[g] = true
	}

	if len(images) == 0 {
		http.Error(w, "no images found", http.StatusNotFound)
		return
	}

	if strings.ToLower(payload.Target) == "s3" && payload.Bucket != "" {
		keyPrefix := strings.TrimSpace(payload.KeyPrefix)
		if keyPrefix != "" && !strings.HasSuffix(keyPrefix, "/") {
			keyPrefix += "/"
		}
		key := fmt.Sprintf("%srestaurant_%d_media_%s.zip", keyPrefix, id, time.Now().Format("20060102T150405"))

		awsCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			http.Error(w, "failed to load AWS config", http.StatusInternalServerError)
			return
		}
		s3client := s3.NewFromConfig(awsCfg)
		uploader := manager.NewUploader(s3client)

		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()
			var imgs []struct {
				Src      string
				ZipName  string
				Kind     string
				Category string
				ItemName string
			}
			for _, it := range images {
				imgs = append(imgs, struct {
					Src      string
					ZipName  string
					Kind     string
					Category string
					ItemName string
				}{Src: it.Src, ZipName: it.ZipName, Kind: it.Kind, Category: it.Category, ItemName: it.ItemName})
			}
			if err := s.writeZipStream(ctx, pw, imgs, manifest); err != nil {
				fmt.Println("error writing zip to pipe:", err)
				_ = pw.CloseWithError(err)
			}
		}()

		putInput := &s3.PutObjectInput{
			Bucket: &payload.Bucket,
			Key:    &key,
		}
		if payload.Public {
			acl := "public-read"
			putInput.ACL = s3.ObjectCannedACL(acl)
		}

		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: putInput.Bucket,
			Key:    putInput.Key,
			Body:   pr,
			ACL:    putInput.ACL,
		})
		if err != nil {
			http.Error(w, "failed to upload to S3: "+err.Error(), http.StatusInternalServerError)
			return
		}

		presigner := s3.NewPresignClient(s3client)
		getInput := &s3.GetObjectInput{
			Bucket: &payload.Bucket,
			Key:    &key,
		}
		presignCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		presignRes, err := presigner.PresignGetObject(presignCtx, getInput, s3.WithPresignExpires(24*time.Hour))
		if err != nil {
			_ = insertAuditLog(ctx, s.store.DB, id, claims["email"].(string), "export_media_s3", map[string]any{"bucket": payload.Bucket, "key": key, "count": len(images)}, r.RemoteAddr)
			writeJSON(w, map[string]any{"ok": true, "s3": fmt.Sprintf("s3://%s/%s", payload.Bucket, key)})
			return
		}

		_ = insertAuditLog(ctx, s.store.DB, id, claims["email"].(string), "export_media_s3", map[string]any{"bucket": payload.Bucket, "key": key, "count": len(images)}, r.RemoteAddr)
		writeJSON(w, map[string]any{"ok": true, "url": presignRes.URL})
		return
	}

	filename := fmt.Sprintf("restaurant_%d_media_%s.zip", id, time.Now().Format("20060102T150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	var imgs []struct {
		Src      string
		ZipName  string
		Kind     string
		Category string
		ItemName string
	}
	for _, it := range images {
		imgs = append(imgs, struct {
			Src      string
			ZipName  string
			Kind     string
			Category string
			ItemName string
		}{Src: it.Src, ZipName: it.ZipName, Kind: it.Kind, Category: it.Category, ItemName: it.ItemName})
	}

	if err := s.writeZipStream(ctx, w, imgs, manifest); err != nil {
		fmt.Println("zip stream error:", err)
	}
	_ = insertAuditLog(ctx, s.store.DB, id, claims["email"].(string), "export_media_download", map[string]any{"count": len(images)}, r.RemoteAddr)
}
