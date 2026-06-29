package v1

import (
	"fmt"
	"io/fs"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/gin-gonic/gin"
)

const defaultSegmentDurationMillis = 60 * 1000

var (
	newUploadRouteDirRegex = regexp.MustCompile(`^([0-9a-fA-F]+)--([0-9a-fA-F]+)--(\d+)$`)
	uploadFileTypeByName   = map[string]string{
		"qcamera.ts":   "qcameras",
		"fcamera.hevc": "cameras",
		"dcamera.hevc": "dcameras",
		"ecamera.hevc": "ecameras",
		"qlog.bz2":     "qlogs",
		"qlog.zst":     "qlogs",
		"rlog.bz2":     "logs",
		"rlog.zst":     "logs",
	}
	uploadResponseKeys = []string{"qlogs", "qcameras", "logs", "cameras", "dcameras", "ecameras"}
)

type uploadedFile struct {
	Segment int
	Name    string
	Type    string
	URL     string
}

type uploadedRouteSummary struct {
	Files     map[string][]string
	Segments  []int
	MaxByType map[string]int
	QCameras  []uploadedFile
}

func GETRouteFile(c *gin.Context) {
	cfg, ok := c.MustGet("config").(*config.Config)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	root, ok := filesystemUploadRoot(cfg)
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
		return
	}

	dongleID := c.Param("dongle_id")
	routeName := c.Param("route")
	segment, err := strconv.Atoi(c.Param("segment"))
	if err != nil || segment < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "segment must be a non-negative integer"})
		return
	}
	fileName := c.Param("filename")
	if _, ok := uploadFileTypeByName[fileName]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file"})
		return
	}

	dirName, ok := findRouteSegmentDir(root, dongleID, routeNameBase(routeName), segment)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	path, err := safeUploadPath(root, dongleID, dirName, fileName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	c.Header("Cache-Control", "private, max-age=3600")
	switch fileName {
	case "qcamera.ts":
		c.Header("Content-Type", "video/mp2t")
	case "fcamera.hevc", "dcamera.hevc", "ecamera.hevc":
		c.Header("Content-Type", "video/hevc")
	default:
		c.Header("Content-Type", "application/octet-stream")
	}
	c.File(path)
}

func summarizeUploadedRoute(cfg *config.Config, dongleID, routeName string) uploadedRouteSummary {
	summary := uploadedRouteSummary{
		Files:    emptyRouteFiles(),
		Segments: []int{},
		MaxByType: map[string]int{
			"qlogs": -1, "qcameras": -1, "logs": -1, "cameras": -1, "dcameras": -1, "ecameras": -1,
		},
		QCameras: []uploadedFile{},
	}

	root, ok := filesystemUploadRoot(cfg)
	if !ok {
		return summary
	}

	deviceRoot, err := safeUploadPath(root, dongleID)
	if err != nil {
		return summary
	}
	entries, err := os.ReadDir(deviceRoot)
	if err != nil {
		return summary
	}

	routeBase := routeNameBase(routeName)
	seenSegments := map[int]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		segment, ok := segmentFromUploadDir(routeBase, entry.Name())
		if !ok {
			continue
		}

		dirPath, err := safeUploadPath(root, dongleID, entry.Name())
		if err != nil {
			continue
		}
		fileEntries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, fileEntry := range fileEntries {
			if fileEntry.IsDir() {
				continue
			}
			typeName, ok := uploadFileTypeByName[fileEntry.Name()]
			if !ok {
				continue
			}
			file := uploadedFile{
				Segment: segment,
				Name:    fileEntry.Name(),
				Type:    typeName,
				URL:     routeFileURL(cfg, dongleID, routeBase, segment, fileEntry.Name()),
			}
			summary.Files[typeName] = append(summary.Files[typeName], file.URL)
			if segment > summary.MaxByType[typeName] {
				summary.MaxByType[typeName] = segment
			}
			if typeName == "qcameras" {
				summary.QCameras = append(summary.QCameras, file)
			}
			if !seenSegments[segment] {
				summary.Segments = append(summary.Segments, segment)
				seenSegments[segment] = true
			}
		}
	}

	sort.Ints(summary.Segments)
	sort.Slice(summary.QCameras, func(i, j int) bool {
		return summary.QCameras[i].Segment < summary.QCameras[j].Segment
	})
	for _, key := range uploadResponseKeys {
		sort.Slice(summary.Files[key], func(i, j int) bool {
			return routeURLSegment(summary.Files[key][i]) < routeURLSegment(summary.Files[key][j])
		})
	}

	return summary
}

func emptyRouteFiles() map[string][]string {
	files := make(map[string][]string, len(uploadResponseKeys))
	for _, key := range uploadResponseKeys {
		files[key] = []string{}
	}
	return files
}

func routeFileURL(cfg *config.Config, dongleID, routeName string, segment int, fileName string) string {
	return fmt.Sprintf("%s/%d/%s", routeFilesBaseURL(cfg, dongleID, routeName), segment, fileName)
}

func routeFilesBaseURL(cfg *config.Config, dongleID, routeName string) string {
	return fmt.Sprintf("%s/v1/route_file/%s/%s",
		strings.TrimRight(cfg.HTTP.BackendURL, "/"),
		dongleID,
		routeName)
}

func routeResponseURL(cfg *config.Config, dongleID string, route models.Route) string {
	if route.URL != "" {
		return route.URL
	}
	return routeFilesBaseURL(cfg, dongleID, route.RouteID)
}

func routeNameBase(routeName string) string {
	if idx := strings.LastIndex(routeName, ":"); idx >= 0 {
		return routeName[:idx]
	}
	return routeName
}

func segmentFromUploadDir(routeBase, dirName string) (int, bool) {
	if matches := newUploadRouteDirRegex.FindStringSubmatch(dirName); len(matches) == 4 && matches[2] == routeBase {
		segment, err := strconv.Atoi(matches[3])
		return segment, err == nil && segment >= 0
	}

	prefix := routeBase + "--"
	if strings.HasPrefix(dirName, prefix) {
		segment, err := strconv.Atoi(strings.TrimPrefix(dirName, prefix))
		return segment, err == nil && segment >= 0
	}

	return 0, false
}

func routeURLSegment(fileURL string) int {
	parts := strings.Split(strings.TrimRight(fileURL, "/"), "/")
	if len(parts) < 2 {
		return -1
	}
	segment, err := strconv.Atoi(parts[len(parts)-2])
	if err != nil {
		return -1
	}
	return segment
}

func filesystemUploadRoot(cfg *config.Config) (string, bool) {
	if cfg.Persistence.Uploads.Driver != config.UploadsDriverFilesystem {
		return "", false
	}
	if cfg.Persistence.Uploads.FilesystemOptions.Directory == "" {
		return "", false
	}
	return cfg.Persistence.Uploads.FilesystemOptions.Directory, true
}

func safeUploadPath(root string, parts ...string) (string, error) {
	for _, part := range parts {
		if part == "" || strings.Contains(part, "/") || part == "." || part == ".." || !fs.ValidPath(part) {
			return "", fmt.Errorf("invalid upload path part: %s", part)
		}
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(append([]string{rootAbs}, parts...)...)
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if pathAbs != rootAbs && !strings.HasPrefix(pathAbs, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("upload path escapes root")
	}
	return pathAbs, nil
}

func findRouteSegmentDir(root, dongleID, routeName string, segment int) (string, bool) {
	deviceRoot, err := safeUploadPath(root, dongleID)
	if err != nil {
		return "", false
	}
	entries, err := os.ReadDir(deviceRoot)
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entrySegment, ok := segmentFromUploadDir(routeName, entry.Name())
		if ok && entrySegment == segment {
			return entry.Name(), true
		}
	}
	return "", false
}

func routeSegmentMetadata(route models.Route, uploadedSegments []int) ([]int64, []int64, []int) {
	segments := make([]int, 0, len(route.SegmentNumbers)+len(uploadedSegments))
	seen := map[int]bool{}
	for _, segment := range route.SegmentNumbers {
		if segment >= 0 && segment <= math.MaxInt32 {
			segmentInt := int(segment)
			segments = append(segments, segmentInt)
			seen[segmentInt] = true
		}
	}
	for _, segment := range uploadedSegments {
		if !seen[segment] {
			segments = append(segments, segment)
			seen[segment] = true
		}
	}
	sort.Ints(segments)
	if len(segments) == 0 {
		segments = []int{0}
	}

	starts := make([]int64, len(segments))
	ends := make([]int64, len(segments))
	for i, segment := range segments {
		if idx := routeSegmentIndex(route, segment); idx >= 0 {
			starts[i] = normalizeAPITime(route.SegmentStartTimes[idx])
			ends[i] = normalizeAPITime(route.SegmentEndTimes[idx])
			if starts[i] > 0 && ends[i] > starts[i] {
				continue
			}
		}

		routeStart := route.StartTime.UnixMilli()
		starts[i] = routeStart + int64(segment*defaultSegmentDurationMillis)
		ends[i] = starts[i] + defaultSegmentDurationMillis
		if !route.EndTime.IsZero() && ends[i] > route.EndTime.UnixMilli() {
			ends[i] = route.EndTime.UnixMilli()
		}
		if ends[i] <= starts[i] {
			ends[i] = starts[i] + defaultSegmentDurationMillis
		}
	}

	return starts, ends, segments
}

func routeSegmentIndex(route models.Route, segment int) int {
	if len(route.SegmentNumbers) != len(route.SegmentStartTimes) || len(route.SegmentNumbers) != len(route.SegmentEndTimes) {
		return -1
	}
	for i, routeSegment := range route.SegmentNumbers {
		if int(routeSegment) == segment {
			return i
		}
	}
	return -1
}

func normalizeAPITime(value int64) int64 {
	if value > 1000000000000000 {
		return value / int64(time.Millisecond)
	}
	return value
}
