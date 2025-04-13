package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
)


type Record struct {
    EventId       string `parquet:"name=EventId, type=BYTE_ARRAY, convertedtype=UTF8"`
    Message       string `parquet:"name=Message, type=BYTE_ARRAY, convertedtype=UTF8"`
    NanoTimeStamp string `parquet:"name=NanoTimeStamp, type=BYTE_ARRAY, convertedtype=UTF8"`
}


type SearchEngine struct {
    records []Record
    index   map[string][]int
}


func main() {
    var parquetDir string
    flag.StringVar(&parquetDir, "parquet-dir", "", "Directory containing Parquet files")
    flag.Parse()

    if parquetDir == "" {
        log.Fatal("Please provide a Parquet directory using --parquet-dir")
    }

    
    files, err := findParquetFiles(parquetDir)
    if err != nil {
        log.Fatalf("Failed to read Parquet directory %s: %v", parquetDir, err)
    }
    if len(files) == 0 {
        log.Fatal("No Parquet files found in directory")
    }

    se := &SearchEngine{
        index: make(map[string][]int),
    }

 
    for _, file := range files {
        log.Printf("Loading file: %s", file)
        records, err := readParquetFile(file)
        if err != nil {
            log.Printf("Failed to read Parquet file %s: %v", file, err)
            continue
        }
        se.addRecords(records)
    }

    log.Printf("Loaded %d records", len(se.records))

  
    http.HandleFunc("/search", se.searchHandler)
    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}


func findParquetFiles(dir string) ([]string, error) {
    var files []string
    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, err
    }
    for _, entry := range entries {
        if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".parquet") {
            files = append(files, filepath.Join(dir, entry.Name()))
        }
    }
    return files, nil
}


func readParquetFile(filePath string) ([]Record, error) {
    fr, err := local.NewLocalFileReader(filePath)
    if err != nil {
        return nil, err
    }
    defer fr.Close()

    pr, err := reader.NewParquetReader(fr, new(Record), 4)
    if err != nil {
        return nil, err
    }
    defer pr.ReadStop()


    for _, elem := range pr.SchemaHandler.SchemaElements {
        if elem.Name != "" {
            log.Printf("Schema field in %s: %s (Type: %v)", filePath, elem.Name, elem.Type)
        }
    }

    num := int(pr.GetNumRows())
    records := make([]Record, num)
    if err = pr.Read(&records); err != nil {
        return nil, err
    }
    return records, nil
}


func (se *SearchEngine) addRecords(newRecords []Record) {
    startIdx := len(se.records)
    for _, rec := range newRecords {
        se.records = append(se.records, rec)
        
        terms := tokenize(rec.Message + " " + rec.EventId + " " + rec.NanoTimeStamp)
        for _, term := range terms {
            se.index[term] = append(se.index[term], startIdx)
        }
        startIdx++
    }
}


func tokenize(text string) []string {
    return strings.Fields(strings.ToLower(text))
}


func (se *SearchEngine) searchHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
        return
    }

    startTime := time.Now()
    terms := tokenize(query)
    if len(terms) == 0 {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "results": []Record{},
            "count":   0,
            "time_ms": 0,
        })
        return
    }

    
    indices := se.index[terms[0]]
    for _, term := range terms[1:] {
        indices = intersect(indices, se.index[term])
    }


    results := make([]Record, 0, len(indices))
    for _, idx := range indices {
        results = append(results, se.records[idx])
    }
    sortResults(results)


    duration := time.Since(startTime).Milliseconds()
    response := map[string]interface{}{
        "results": results,
        "count":   len(results),
        "time_ms": duration,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}


func intersect(a, b []int) []int {
    m := make(map[int]bool)
    for _, item := range a {
        m[item] = true
    }
    var result []int
    for _, item := range b {
        if m[item] {
            result = append(result, item)
        }
    }
    return result
}


func sortResults(results []Record) {
    sort.Slice(results, func(i, j int) bool {
        
        ti, _ := strconv.ParseInt(results[i].NanoTimeStamp, 10, 64)
        tj, _ := strconv.ParseInt(results[j].NanoTimeStamp, 10, 64)
        return ti > tj
    })
}