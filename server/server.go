package server

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	Name     string
	Status   string
	Children map[string]*Node
}

func setupRoutes() {
	http.HandleFunc("/drift-detector/report", reportHandler)
	http.HandleFunc("/drift-detector/download-reports", downloadReportsHandler)
	http.Handle("/drift-detector/metrics", promhttp.Handler())
	http.Handle("/drift-detector/static/", http.StripPrefix("/drift-detector/static/", http.FileServer(http.Dir("./static"))))
}

func Run() {
	setupRoutes()
	log.Info("server is running on port 8080")
	http.ListenAndServe(":8080", nil)
}

const closedFolderIcon = `<i class='fas fa-folder'></i>`

// const openFolderIcon = `<i class='fas fa-folder-open'></i>`

func renderNode(node *Node, depth int) string {
	if node == nil {
		return ""
	}

	statusColor := ""
	folderColor := ""
	switch node.Status {
	case "error":
		statusColor = "color:red;"
		folderColor = "background-color:#ffd5d5;" // light red
	case "drifted":
		statusColor = "color:black;"              // Using black text for yellow background for better readability
		folderColor = "background-color:#fff3cd;" // light yellow
	case "No changes":
		statusColor = "color:green;"
		folderColor = "background-color:#d4edda;" // light green
	default:
		folderColor = "background-color:white;"
	}

	indentation := fmt.Sprintf("margin-left:%dpx;", depth*10) // 10px of indentation for each depth level

	// Determine the display property for the children div. If depth is 0 (top-level), it's shown by default.
	displayProperty := "none"
	if depth == 0 {
		displayProperty = "block"
	}

	var result string
	if depth > 0 || (depth == 0 && node.Status != "") {
		result = fmt.Sprintf(`<div style="%s;cursor:pointer;%s" onclick="toggleChildren(event)">%s %s <span style="%s"> %s</span></div>`, indentation, folderColor, closedFolderIcon, node.Name, statusColor, node.Status)
	} else {
		result = fmt.Sprintf(`<div style="%s;cursor:pointer;%s" onclick="toggleChildren(event)">%s %s</div>`, indentation, folderColor, closedFolderIcon, node.Name)
	}

	if len(node.Children) > 0 {
		result += fmt.Sprintf(`<div style="display:%s;">`, displayProperty)
		for _, child := range node.Children {
			result += renderNode(child, depth+1)
		}
		result += `</div>`
	}

	return result
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir("csv/data/")
	if err != nil {
		http.Error(w, "Failed to list CSV files", http.StatusInternalServerError)
		return
	}

	// Create a unified root node
	unifiedRoot := &Node{
		Name:     "chainstack",
		Children: make(map[string]*Node),
	}
	var totalErrorCount, totalDriftedCount, totalNoChangesCount int

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".csv") {
			root, errorCount, driftedCount, noChangesCount, err := ReadCSVToNodes("csv/data/" + file.Name())
			if err != nil {
				continue
			}

			mergeTrees(unifiedRoot, root)
			totalErrorCount += errorCount
			totalDriftedCount += driftedCount
			totalNoChangesCount += noChangesCount
		}
	}

	chartData := fmt.Sprintf(`
        <script>
            let errorCount = %d;
            let driftedCount = %d;
            let noChangesCount = %d;
        </script>
    `, totalErrorCount, totalDriftedCount, totalNoChangesCount)

	allData := renderNode(unifiedRoot, 0)

	// Read the HTML template
	htmlBytes, err := ioutil.ReadFile("static/report.html")
	if err != nil {
		http.Error(w, "Failed to load HTML", http.StatusInternalServerError)
		return
	}

	// Replace the placeholder with the rendered data
	htmlStr := strings.Replace(string(htmlBytes), "{{DATA_PLACEHOLDER}}", allData, 1) + chartData + "</body></html>"

	w.Write([]byte(htmlStr))
}

// mergeTrees will merge src into dest recursively
func mergeTrees(dest, src *Node) {
	for name, srcChild := range src.Children {
		if destChild, exists := dest.Children[name]; exists {
			mergeTrees(destChild, srcChild)
		} else {
			dest.Children[name] = srcChild
		}
	}
}

func ReadCSVToNodes(filepath string) (*Node, int, int, int, error) {
	errorCount := 0
	driftedCount := 0
	noChangesCount := 0

	file, err := os.Open(filepath)
	if err != nil {
		return nil, errorCount, driftedCount, noChangesCount, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		return nil, errorCount, driftedCount, noChangesCount, err
	}

	root := &Node{Children: make(map[string]*Node)}

	for _, record := range records {
		current := root
		paths := strings.Split(record[0], "/")
		status := record[1]
		if status == "error" {
			errorCount++
		} else if status == "drifted" {
			driftedCount++
		} else if status == "No changes" {
			noChangesCount++
		}
		for i, path := range paths {
			if current.Children[path] == nil {
				current.Children[path] = &Node{
					Name:     path,
					Children: make(map[string]*Node),
				}
			}
			current = current.Children[path]
			if i == len(paths)-1 && path != "prod" && path != "dev" {
				current.Status = status
			}
		}
	}

	return root, errorCount, driftedCount, noChangesCount, nil
}

func downloadReportsHandler(w http.ResponseWriter, r *http.Request) {
	folderPath := "/csv/data/"
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		http.Error(w, "Failed to read the directory", http.StatusInternalServerError)
		return
	}

	var filePaths []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".csv") {
			filePaths = append(filePaths, folderPath+file.Name())
		}
	}

	zipName := "reports.zip"
	if err := ZipFiles(zipName, filePaths); err != nil {
		http.Error(w, "Failed to zip the files", http.StatusInternalServerError)
		log.Warnf("error zipping files: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+zipName)
	http.ServeFile(w, r, zipName)
}

func ZipFiles(zipName string, files []string) error {
	newZipFile, err := os.Create(zipName)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	writer := zip.NewWriter(newZipFile)
	defer writer.Close()

	for _, file := range files {
		fileToZip, err := os.Open(file)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

		info, err := fileToZip.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Method = zip.Deflate

		writer, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, fileToZip)
		if err != nil {
			return err
		}
	}
	return nil
}
