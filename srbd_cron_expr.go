package main

import (
"fmt"
"os"
"encoding/csv"
"net/http"
"io/ioutil"
"strings"
)

func main() {
	matrix := [][]string{
		[]string{"dp.esimo.aari.ru", "RU_AARI_1103", "RU_AARI_1104", "RU_AARI_1121", "RU_AARI_1122",
									"RU_AARI_1132", "RU_AARI_1133", "RU_AARI_1135", "RU_AARI_1136",
									"RU_AARI_1137", "RU_AARI_1140", "RU_AARI_1142", "RU_AARI_1143",
									"RU_AARI_1144", "RU_AARI_1145", "RU_AARI_1146", "RU_AARI_1147",
									"RU_AARI_1148", "RU_AARI_1149", "RU_AARI_1150", "RU_AARI_3101",
									"RU_AARI_3103", "RU_AARI_3151", "RU_AARI_3330", "RU_AARI_3430"},
		[]string{"dp.hydrometcentre.esimo.ru:8080", "RU_Hydrometcentre_42", "RU_Hydrometcentre_46", 
		"RU_Hydrometcentre_47", "RU_Hydrometcentre_53", "RU_Hydrometcentre_60",
							"RU_Hydrometcentre_61", "RU_Hydrometcentre_62", "RU_Hydrometcentre_63",
							"RU_Hydrometcentre_64", "RU_Hydrometcentre_65", "RU_Hydrometcentre_66",
							"RU_Hydrometcentre_68", "RU_Hydrometcentre_69", "RU_Hydrometcentre_122"},
		[]string{"dpms.meteo.ru", "RU_RIHMI-WDC_67", "RU_RIHMI-WDC_1196", "RU_RIHMI-WDC_1198",
							"RU_RIHMI-WDC_1172", "RU_RIHMI-WDC_1197", "RU_RIHMI-WDC_1242",
							"RU_RIHMI-WDC_1195", "RU_RIHMI-WDC_1160", "RU_RIHMI-WDC_1161","RU_RIHMI-WDC_1164",
							"RU_RIHMI-WDC_1167", "RU_RIHMI-WDC_1170", "RU_RIHMI-WDC_1172", "RU_RIHMI-WDC_1195",
							"RU_RIHMI-WDC_1196", "RU_RIHMI-WDC_2897", "RU_RIHMI-WDC_2901", "RU_RIHMI-WDC_2900",
							"RU_RIHMI-WDC_227", "RU_RIHMI-WDC_487", "RU_RIHMI-WDC_489", "RU_RIHMI-WDC_490",
							"RU_RIHMI-WDC_492", "RU_RIHMI-WDC_67", "RU_RIHMI-WDC_1197", "RU_RIHMI-WDC_1242",
							"RU_RIHMI-WDC_1326"},
		[]string{"dp.typhoon.esimo.ru", "RU_Typhoon_01", "RU_Typhoon_24"},		
	}

	csvfile, err_csv := os.Create("ir_cron_times.csv")
	if err_csv != nil {
		panic("Error creating CSV file")
	}
	defer csvfile.Close()

	writer := csv.NewWriter(csvfile)
	writer.Write([]string{"sep=,"})

	for i := 0; i < len(matrix); i++ {
		for j := 1; j <= len(matrix[i][1:]); j++ {
			domain := matrix[i][0]
			resource := matrix[i][j]
			fmt.Print(resource + ",")

			cronExpression := getCronExpression(domain, resource)
			fmt.Print(cronExpression + ",");

			startTime := getCronStartTime(cronExpression)
			fmt.Println(startTime)

			writer.Write([]string{resource, startTime})
		}
	}

	writer.Flush()
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
			text += " (every " + freq + " hour(s))"
		}

		return text;
	}

	return ""
}