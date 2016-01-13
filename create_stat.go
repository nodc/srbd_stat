package main

import (
"fmt"
"os"
"net/http"
"io/ioutil"
"encoding/xml"
"encoding/csv"
"database/sql"
_ "github.com/lib/pq"
"time"
)

const (
	DB_ADDR		= "10.1.91.238:5432"
    DB_USER     = "bid"
    DB_PASSWORD = "bidesimo"
    DB_NAME     = "bid"
)

type E2ETemporalExtent struct {
	XMLName xml.Name `xml:"E2ETemporalExtent"`
	BeginDateTime string `xml:"beginDateTime"`
	EndDateTime string `xml:"endDateTime"`
}

type E2ESearchMD struct {
	XMLName xml.Name `xml:"E2ESearchMD"`
	TemporalExtent E2ETemporalExtent `xml:"E2ETemporalExtent"`
}

type Root struct {
	XMLName xml.Name `xml:"root"`	
	Metadata E2ESearchMD
}

func main() {
	row := 21;
	col := 4;
	biddata := make([][]string, row)
	for i := range biddata {
		biddata[i] = make([]string, col)
	}

	dbinfo := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
        DB_USER, DB_PASSWORD, DB_ADDR, DB_NAME)
	fmt.Println(dbinfo)
	db, err_db := sql.Open("postgres", dbinfo)
	if err_db != nil {
		fmt.Printf("Coudn't connect to BID: %s", err_db)
	}
    defer db.Close()

    rows, err_select := db.Query("select * from monit.monit_report")
    if err_select != nil {
    	fmt.Printf("Coudn't run query to BID: %s", err_select)
    }

    i := 0;
    for (rows.Next()) {
    	var resourceId string
    	var bidupdated time.Time
    	var beginDateTime string
    	var endDateTime string

    	err_fetch := rows.Scan(&resourceId, &bidupdated, &beginDateTime, &endDateTime)
    	if err_fetch != nil {
    		fmt.Printf("Coudn't fetch data from BID: %s", err_fetch)
    	}

    	fmt.Println(resourceId + ";" + bidupdated.Format(time.RFC3339) + ";" + beginDateTime + ";" + endDateTime)
    	biddata[i] = []string{resourceId, bidupdated.Format(time.RFC3339), beginDateTime, endDateTime}
    	i++;
    }

    fmt.Println("biddata: " + biddata[0][0])
    fmt.Println("biddata[1]: " + biddata[0][1])

	matrix := [][]string{
		[]string{"dp.hydrometcentre.esimo.ru:8080", "RU_Hydrometcentre_42", "RU_Hydrometcentre_46", "RU_Hydrometcentre_60",
							"RU_Hydrometcentre_61", "RU_Hydrometcentre_62", "RU_Hydrometcentre_63",
							"RU_Hydrometcentre_64", "RU_Hydrometcentre_65", "RU_Hydrometcentre_66",
							"RU_Hydrometcentre_68", "RU_Hydrometcentre_69", "RU_Hydrometcentre_122"},
		[]string{"dpms.meteo.ru", "RU_RIHMI-WDC_67", "RU_RIHMI-WDC_1196", "RU_RIHMI-WDC_1198",
							"RU_RIHMI-WDC_1172", "RU_RIHMI-WDC_1197", "RU_RIHMI-WDC_1242",
							"RU_RIHMI-WDC_1195"},
	}

	csvfile, err_csv := os.Create("ir_stat.csv")
	if err_csv != nil {
		panic("Error creating CSV file")
	}
	defer csvfile.Close()

	is_stat, err_is := os.Open("data_csv.txt")
	if err_is != nil {
		fmt.Println("Error during reading IS report");
	}
	defer is_stat.Close()

	is_reader := csv.NewReader(is_stat)
	is_report, _ := is_reader.ReadAll()

	writer := csv.NewWriter(csvfile)
	writer.Write([]string{"sep=,"})
	writer.Write([]string{"Идентификатор ИР", "ПД", "СИ", "БИД (время обновления)", "БИД (метаданные)"})

	// slices
	for i := 0; i < len(matrix); i++ {
		addr := "http://" + matrix[i][0] + "/dpms/controller?action=getResourceCache&resourceId=";
		fmt.Println(addr)

		for j := 1; j <= len(matrix[i][1:]); j++ {
			resource := matrix[i][j]
			fmt.Println(resource)

			writer.Write([]string{resource, "", "", "", ""})

			res, err := http.Get(addr + resource)
			if err != nil {
				panic(err.Error())
			}	

			body, err := ioutil.ReadAll(res.Body)

			var root Root
			err_parse := xml.Unmarshal([]byte(body), &root)
			if err_parse != nil {
				fmt.Printf("error: %root", err_parse)
			}
	
			beginDateTime := root.Metadata.TemporalExtent.BeginDateTime
			endDateTime := root.Metadata.TemporalExtent.EndDateTime

			fmt.Println(beginDateTime + " - " + endDateTime)

			// поиск по кешу СИ
			var is_data_time string
			for _, is_report_record := range is_report {
				if (is_report_record[0] == (resource + "_1.nc")) {
					is_data_time = is_report_record[2]
				}
			}

			// поиск по БИД
			var bid_update_time string
			var bid_md_begin string
			var bid_md_end string
			for z := range biddata {
				if resource == biddata[z][0] {
					bid_update_time = biddata[z][1]
					bid_md_begin = biddata[z][2]
					bid_md_end = biddata[z][3]
				}
			}

			writer.Write([]string{"метаданные", beginDateTime + " - " + endDateTime, "", 
				bid_update_time, bid_md_begin + "-" + bid_md_end})
			writer.Write([]string{"данные", "", is_data_time, "", ""})
		}

		writer.Flush()
	}
}