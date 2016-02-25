package main

import (
"fmt"
"os"
"net/http"
"net/http/cookiejar"
"io/ioutil"
"encoding/xml"
"encoding/csv"
"database/sql"
_ "github.com/lib/pq"
"time"
"log"
"strings"
"sort"
"strconv"
"github.com/tealeg/xlsx"
"github.com/BurntSushi/toml"
)

type Config struct {
	DB_ADDR	string
    DB_USER string
    DB_PASSWORD string
    DB_NAME     string
    Is_date_pattern string
	Global_date_pattern string
}

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

// GIS WMS getCapabilities()
type WMS_Layer struct {
	XMLName xml.Name `xml:"Layer"`
	Title string `xml:"Title"`
}

type InstanceLayer struct {
	XMLName xml.Name `xml:"Layer"`
	Layer []WMS_Layer `xml:"Layer"`
}

type Capability struct {
	XMLName xml.Name `xml:"Capability"`
	Instance InstanceLayer `xml:"Layer"`
}

type WMS_Capabilities struct {
	XMLName xml.Name `xml:"WMS_Capabilities"`
	Cap Capability `xml:"Capability"`
}

func main() {
	var config = ReadConfig()

	row := 21;
	col := 6;
	biddata := make([][]string, row)
	for i := range biddata {
		biddata[i] = make([]string, col)
	}

	var file *xlsx.File
    // var sheet *xlsx.Sheet
    // var excel_row *xlsx.Row
    // var cell *xlsx.Cell
    var err_excel error

    file = xlsx.NewFile()
    _, err_excel = file.AddSheet("Sheet1")
    if err_excel != nil {
        fmt.Printf(err_excel.Error())
    }

	dbinfo := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
        config.DB_USER, config.DB_PASSWORD, config.DB_ADDR, config.DB_NAME)
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
    	var bid_data_min string
		var bid_data_max string

    	err_fetch := rows.Scan(&resourceId, &bidupdated, &beginDateTime, &endDateTime, 
    							&bid_data_min, &bid_data_max)
    	if err_fetch != nil {
    		fmt.Printf("Coudn't fetch data from BID: %s", err_fetch)
    	}

    	biddata[i] = []string{resourceId, bidupdated.Format(time.RFC3339), beginDateTime, endDateTime,
    							bid_data_min, bid_data_max}
    	i++;
    }

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
	t := time.Now()
	title := t.Format(time.RFC850) + "\n"
	writer.Write([]string{"Время генерации справки: " + title})
	writer.Write([]string{"Идентификатор ИР", "ПД", "СИ", "БИД (время обновления)", "БИД", "ГИС"})

	// slices
	for i := 0; i < len(matrix); i++ {
		addr := "http://" + matrix[i][0] + "/dpms/controller?action=getResourceCache&resourceId=";		

		for j := 1; j <= len(matrix[i][1:]); j++ {
			resource := matrix[i][j]
			fmt.Println(resource)

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

			// get min/max dates from GIS
			layer_min_date, layer_max_date := getWMSLayersDates(resource)
			var layer_temporal string

			if (layer_min_date == layer_max_date) {
				layer_temporal = layer_min_date
			} else if (layer_min_date == "") {
				layer_temporal = layer_max_date
			} else if (layer_max_date == "") {
				layer_temporal = layer_min_date
			} else {
				layer_temporal = layer_min_date + " - " + layer_max_date
			}

			cronExpression := getCronExpression(matrix[i][0], resource);
			cronExpression = "запуск в " + getCronStartTime(cronExpression)
			
			// ид ИР, ПД, СИ, БИД (время обновления), БИД, ГИС 
			writer.Write([]string{resource, cronExpression, "", "", "", "", "", ""})

			// поиск по кешу СИ			
			var is_data_time string

			for _, is_report_record := range is_report {
				if (is_report_record[0] == (resource + "_1.nc")) {
					is_data_time = is_report_record[2]
					is_date, _ := time.Parse(config.Is_date_pattern, is_data_time)
					fmt.Println(is_date)
					is_data_time = is_date.Format(config.Global_date_pattern)
				}
			}

			// поиск по БИД
			var bid_update_time string
			var bid_md_begin string
			var bid_md_end string
			var bid_data_min string
			var bid_data_max string

			for z := range biddata {
				if resource == biddata[z][0] {
					bid_update_time = biddata[z][1]
					bid_md_begin = biddata[z][2]
					bid_md_end = biddata[z][3]
					bid_data_min = biddata[z][4]
					bid_data_max = biddata[z][5]
				}
			}

			bid_temporal := bid_data_min + "-" + bid_data_max

			writer.Write([]string{"метаданные", beginDateTime + "-" + endDateTime, "", 
				bid_update_time, bid_md_begin + "-" + bid_md_end, ""})
			writer.Write([]string{"данные", "", is_data_time, "", bid_temporal, layer_temporal})
		}

		writer.Flush()
	}
}

// returns min date, max date from layer titles
func getWMSLayersDates (resourceId string) (string, string) {
	addr := "http://gis.esimo.ru/resources/" + resourceId + "/wms?request=GetCapabilities"
	fmt.Println(addr)
	cookieJar, _ := cookiejar.New(nil) 

	client := &http.Client{ 
		Jar: cookieJar, 
	} 

	req, err := http.NewRequest("GET", addr, nil) 

	if err != nil { 
		log.Fatalln(err) 
	} 

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.106 Safari/537.36") 

	resp, err := client.Do(req) 
	if err != nil { 
		log.Fatalln(err) 
	} 

	defer resp.Body.Close() 
	body, err := ioutil.ReadAll(resp.Body) 
	if err != nil { 
		log.Fatalln(err) 
	} 

	year := strconv.Itoa(time.Now().Year())

	if (len(body) > 5 && strings.HasPrefix(string(body), "<?xml")) {
		var layer WMS_Capabilities
		err_parse := xml.Unmarshal([]byte(body), &layer)
		if err_parse != nil {
			fmt.Printf("Couldn't unmarshal WMS Layer capabilities", err_parse)
		}

		// массив с датами из тайтлов слоев
		layers_pub_dates := make([]string, len(layer.Cap.Instance.Layer))
		//var layers_pub_dates [len(layer.Cap.Instance.Layer)]string

		for l := range layer.Cap.Instance.Layer {
			index := strings.LastIndex(layer.Cap.Instance.Layer[l].Title, year)
			// длина даты и срока в тайтле слоя = 13			
			if index != -1 && len(layer.Cap.Instance.Layer[l].Title) > index + 14 {
				// layerDate exmaple: 2016-01-26 06ч
				// букву ч убираем
				layerDate := layer.Cap.Instance.Layer[l].Title[index:index + 13]			
				layers_pub_dates[l] = layerDate + "ч"
			}
		}
		var layer_min_date string
		var layer_max_date string

		// сортировка дат из тайтлов слоев
		if (len(layers_pub_dates) > 0) {
			sort.Strings(layers_pub_dates)

			layer_min_date = layers_pub_dates[0]
			layer_max_date = layers_pub_dates[len(layers_pub_dates) - 1]
		}

		return layer_min_date, layer_max_date
	}

	return "", ""
}

func getCronExpression(domain string, resource string) (string) {
	addr := "http://" + domain + "/dpms/controller?action=getCronTriggerExpression&resourceId=";
	res, err := http.Get(addr + resource)
	if err != nil {
		fmt.Printf("Error while getting getCronExpression: %s", err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Couldn't read cron expression from response: %s", err.Error())
	}

	return string(body)
}

// приводит время старта по крону в читабельный вид
func getCronStartTime(expr string) (string) {
	words := strings.Fields(expr)

	if len(words) > 3 {
		var freq string
		var hours string
		if strings.Contains(words[2], "/") {
			freq_index := strings.Index(words[2], "/")
			if freq_index != -1 {
				hours = words[2][0:freq_index]
				freq = words[2][freq_index + 1:]
			}
		} else {
			hours = words[2]
		}

		text := hours + ":" + words[1]

		if freq != "" {
			text += " (каждые " + freq + " час(ов))"
		}

		return text;
	}

	return ""
}

// загрузка конфиг-файла
func ReadConfig() Config {
	configfile := "config.toml"
	_, err := os.Stat(configfile)
	if err != nil {
		log.Fatal("Config file is missing: ", configfile)
	}

	var config Config
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		log.Fatal(err)
	}
	//log.Print(config.Index)
	return config
}