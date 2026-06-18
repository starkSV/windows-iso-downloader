package main

import "strings"

type Product struct {
	ID   string
	Name string
}

type EvalProduct struct {
	Slug    string
	Name    string
	EvalURL string
}

var consumerProducts = []Product{
	{"52", "Windows 8.1 (9600.17415)"},
	{"2378", "Windows 10 22H2 Home China (19045.2006)"},
	{"2618", "Windows 10 22H2 v1 (19045.2965)"},
	{"3113", "Windows 11 24H2 (26100.1742)"},
	{"3114", "Windows 11 24H2 Home China (26100.1742)"},
	{"3115", "Windows 11 24H2 Pro China (26100.1742)"},
	{"3131", "Windows 11 Arm64 24H2 (26100.1742)"},
	{"3132", "Windows 11 Arm64 24H2 Home China (26100.1742)"},
	{"3133", "Windows 11 Arm64 24H2 Pro China (26100.1742)"},
	{"3262", "Windows 11 25H2 (26200.6584)"},
	{"3263", "Windows 11 25H2 Home China (26200.6584)"},
	{"3264", "Windows 11 25H2 Pro China (26200.6584)"},
	{"3265", "Windows 11 Arm64 25H2 (26200.6584)"},
	{"3266", "Windows 11 Arm64 25H2 Home China (26200.6584)"},
	{"3267", "Windows 11 Arm64 25H2 Pro China (26200.6584)"},
	{"3321", "Windows 11 25H2 (Updated Oct)"},
	{"3324", "Windows 11 Arm64 25H2 (Updated Oct)"},
}

var evalProducts = []EvalProduct{
	{"server-2025", "Windows Server 2025", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2025"},
	{"server-2022", "Windows Server 2022", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2022"},
	{"server-2019", "Windows Server 2019", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2019"},
	{"server-2016", "Windows Server 2016", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2016"},
	{"win11-ent", "Windows 11 Enterprise", "https://www.microsoft.com/en-us/evalcenter/download-windows-11-enterprise"},
}

func findProductByID(id string) (Product, bool) {
	for _, p := range consumerProducts {
		if p.ID == id {
			return p, true
		}
	}
	return Product{}, false
}

func searchProducts(query string) []Product {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return consumerProducts
	}
	var results []Product
	for _, p := range consumerProducts {
		name := strings.ToLower(p.Name)
		match := true
		for _, w := range words {
			if !strings.Contains(name, w) {
				match = false
				break
			}
		}
		if match {
			results = append(results, p)
		}
	}
	return results
}

func findEvalProduct(slug string) (EvalProduct, bool) {
	for _, p := range evalProducts {
		if p.Slug == slug {
			return p, true
		}
	}
	return EvalProduct{}, false
}
