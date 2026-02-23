package assets

import (
	"image"
	"path/filepath"
)

// Manager handles asset paths and configurations
type Manager struct {
	AssetsDir   string
	RenderedDir string
	GraphicsDir string
}

// NewManager creates a new asset manager
func NewManager(workDir string) *Manager {
	return &Manager{
		AssetsDir:   filepath.Join(workDir, "assets"),
		RenderedDir: filepath.Join(workDir, "rendered"),
		GraphicsDir: filepath.Join(workDir, "graphics"),
	}
}

// ScrapeTarget defines a web scraping target
type ScrapeTarget struct {
	Name       string
	URL        string
	Selector   string
	OutputPath string
	WaitTime   int // milliseconds
}

// DownloadTarget defines an image download target
type DownloadTarget struct {
	Name       string
	URL        string
	OutputPath string
}

// Asset defines an image asset with crop/resize parameters
type Asset struct {
	Name       string
	InputPath  string
	OutputPath string
	CropRect   image.Rectangle
	TargetSize image.Point
}

// CompositeLayer defines a layer in the composite image
type CompositeLayer struct {
	ImagePath string
	Position  image.Point
}

// GetDownloadTargets returns all download targets
func (m *Manager) GetDownloadTargets() []DownloadTarget {
	return []DownloadTarget{
		{
			Name:       "Stevens Pass Jupiter",
			URL:        "https://streamer8.brownrice.com/cam-images/stevenspassjupiter.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "stevenspassjupiter.jpg"),
		},
		{
			Name:       "Stevens Pass Skyline",
			URL:        "https://streamer8.brownrice.com/cam-images/stevenspassskyline.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "stevenspassskyline.jpg"),
		},
		{
			Name:       "Stevens Pass School",
			URL:        "https://streamer3.brownrice.com/cam-images/stevenspassschool.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "stevenspassschool.jpg"),
		},
		{
			Name:       "WSDOT E Stevens Summit",
			URL:        "https://images.wsdot.wa.gov/nc/002vc06458.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "wsdot_e_stevens_summit.jpg"),
		},
		{
			Name:       "GOES18 North Pacific",
			URL:        "https://cdn.star.nesdis.noaa.gov/GOES18/ABI/SECTOR/np/GEOCOLOR/latest.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "GOES18_north_pacific.jpg"),
		},
		{
			Name:       "WSDOT W Stevens",
			URL:        "https://images.wsdot.wa.gov/nc/002vc06190.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "wsdot_w_stevens.jpg"),
		},
		{
			Name:       "WSDOT US2 Skykomish",
			URL:        "https://images.wsdot.wa.gov/nw/002vc04558.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "wsdot_us2_skykomish.jpg"),
		},
		{
			Name:       "WSDOT Big Windy",
			URL:        "https://images.wsdot.wa.gov/nc/002vc06300.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "wsdot_big_windy.jpg"),
		},
		{
			Name:       "Stevens Pass Snow Stake",
			URL:        "https://streamer8.brownrice.com/cam-images/stevenspasssnowstake.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "stevenspasssnowstake.jpg"),
		},
		{
			Name:       "Stevens Pass Courtyard",
			URL:        "https://player.brownrice.com/snapshot/stevenspasscourtyard",
			OutputPath: filepath.Join(m.AssetsDir, "stevenspasscourtyard.jpg"),
		},
		{
			Name:       "WSDOT Stevens Pass",
			URL:        "https://images.wsdot.wa.gov/nc/002vc06430.jpg",
			OutputPath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass.jpg"),
		},
	}
}

// GetScrapeTargets returns all web scraping targets
func (m *Manager) GetScrapeTargets() []ScrapeTarget {
	return []ScrapeTarget{
		{
			Name:       "Weather.gov Hourly Forecast",
			URL:        "https://forecast.weather.gov/MapClick.php?lat=47.7456&lon=-121.0892&unit=0&lg=english&FcstType=graphical",
			Selector:   "img[src*=\"meteograms/Plotter.php\"]",
			OutputPath: filepath.Join(m.AssetsDir, "weather_gov_hourly_forecast.png"),
			WaitTime:   5000,
		},
		{
			Name:       "Weather.gov Extended Forecast",
			URL:        "https://forecast.weather.gov/MapClick.php?lat=47.7456&lon=-121.0892",
			Selector:   "#seven-day-forecast",
			OutputPath: filepath.Join(m.AssetsDir, "weather_gov_extended_forecast.png"),
			WaitTime:   1000,
		},
		{
			Name:       "NWAC Stevens Observations",
			URL:        "https://nwac.us/data-portal/graph/21/",
			Selector:   "#post-146 > div",
			OutputPath: filepath.Join(m.AssetsDir, "nwac_stevens_observations.png"),
			WaitTime:   15000, // Increased for slow NWAC site
		},
		{
			Name:       "NWAC Avalanche Forecast",
			URL:        "https://nwac.us/avalanche-forecast/#/stevens-pass",
			Selector:   "#nac-tab-resizer > div > div:nth-child(1) > div > div.nac-danger.nac-mb-4 > div.nac-row > div.nac-dangerToday.nac-col-lg-8.nac-mb-3 > div.nac-dangerGraphic",
			OutputPath: filepath.Join(m.AssetsDir, "nwac_stevens_avalanche_forcast.png"),
			WaitTime:   15000, // Increased for slow NWAC site
		},
		{
			Name:       "NWAC Avalanche Forecast Map",
			URL:        "https://nwac.us",
			Selector:   "#danger-map-widget",
			OutputPath: filepath.Join(m.AssetsDir, "nwac_avalanche_forcast.png"),
			WaitTime:   15000, // Increased for slow NWAC site
		},
	}
}

// GetWSDOTHTMLTarget returns the WSDOT pass status HTML extraction target
func (m *Manager) GetWSDOTHTMLTarget() ScrapeTarget {
	return ScrapeTarget{
		Name:       "WSDOT Stevens Pass Status",
		URL:        "https://wsdot.com/travel/real-time/mountainpasses/stevens",
		Selector:   ".full-width.column-container.mountain-pass .column-1",
		OutputPath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass.html"),
		WaitTime:   10000, // 10 seconds for Vue.js page to fully render
	}
}

// GetCropAssets returns all assets that need cropping and resizing
func (m *Manager) GetCropAssets() []Asset {
	return []Asset{
		{
			Name:       "Background Satellite",
			InputPath:  filepath.Join(m.AssetsDir, "GOES18_north_pacific.jpg"),
			OutputPath: filepath.Join(m.AssetsDir, "background_s.jpg"),
			CropRect:   image.Rect(0, 0, 7200, 4050),
			TargetSize: image.Point{X: 3840, Y: 2160},
		},
		{
			Name:       "NWAC Avalanche Forecast Map",
			InputPath:  filepath.Join(m.AssetsDir, "nwac_avalanche_forcast.png"),
			OutputPath: filepath.Join(m.AssetsDir, "nwac_avalanche_forcast_s.jpg"),
			CropRect:   image.Rect(65, 110, 465, 630),
			TargetSize: image.Point{X: 400, Y: 520},
		},
		{
			Name:       "NWAC Stevens Observations",
			InputPath:  filepath.Join(m.AssetsDir, "nwac_stevens_observations.png"),
			OutputPath: filepath.Join(m.AssetsDir, "nwac_stevens_observations_s.jpg"),
			CropRect:   image.Rect(0, 0, 1140, 1439),
			TargetSize: image.Point{X: 855, Y: 1079},
		},
		{
			Name:       "Stevens Pass Courtyard",
			InputPath:  filepath.Join(m.AssetsDir, "stevenspasscourtyard.jpg"),
			OutputPath: filepath.Join(m.AssetsDir, "stevenspasscourtyard_s.jpg"),
			CropRect:   image.Rect(0, 0, 1920, 1080),
			TargetSize: image.Point{X: 680, Y: 382},
		},
		{
			Name:       "Stevens Pass Snow Stake",
			InputPath:  filepath.Join(m.AssetsDir, "stevenspasssnowstake.jpg"),
			OutputPath: filepath.Join(m.AssetsDir, "stevenspasssnowstake_s.jpg"),
			CropRect:   image.Rect(0, 0, 1920, 1080),
			TargetSize: image.Point{X: 680, Y: 382},
		},
		{
			Name:       "Weather.gov Extended Forecast",
			InputPath:  filepath.Join(m.AssetsDir, "weather_gov_extended_forecast.png"),
			OutputPath: filepath.Join(m.AssetsDir, "weather_gov_extended_forecast_s.jpg"),
			CropRect:   image.Rect(0, 100, 1146, 400),
			TargetSize: image.Point{X: 1146, Y: 300},
		},
		{
			Name:       "Weather.gov Hourly Forecast",
			InputPath:  filepath.Join(m.AssetsDir, "weather_gov_hourly_forecast.png"),
			OutputPath: filepath.Join(m.AssetsDir, "weather_gov_hourly_forecast_s.jpg"),
			CropRect:   image.Rect(0, 0, 800, 871),
			TargetSize: image.Point{X: 855, Y: 930},
		},
		{
			Name:       "WSDOT Stevens Pass (Big)",
			InputPath:  filepath.Join(m.AssetsDir, "wsdot_stevens_pass.jpg"),
			OutputPath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass_b.jpg"),
			CropRect:   image.Rect(0, 0, 400, 225),
			TargetSize: image.Point{X: 400, Y: 225},
		},
		{
			Name:       "Stevens Pass Jupiter (Scaled)",
			InputPath:  filepath.Join(m.AssetsDir, "stevenspassjupiter.jpg"),
			OutputPath: filepath.Join(m.AssetsDir, "stevenspassjupiter_s.jpg"),
			CropRect:   image.Rect(0, 0, 1280, 720),
			TargetSize: image.Point{X: 1075, Y: 605},
		},
		{
			Name:       "Stevens Pass Skyline (Scaled)",
			InputPath:  filepath.Join(m.AssetsDir, "stevenspassskyline.jpg"),
			OutputPath: filepath.Join(m.AssetsDir, "stevenspassskyline_s.jpg"),
			CropRect:   image.Rect(0, 0, 1280, 720),
			TargetSize: image.Point{X: 1075, Y: 605},
		},
		{
			Name:       "Stevens Pass School (Scaled)",
			InputPath:  filepath.Join(m.AssetsDir, "stevenspassschool.jpg"),
			OutputPath: filepath.Join(m.AssetsDir, "stevenspassschool_s.jpg"),
			CropRect:   image.Rect(0, 0, 1280, 720),
			TargetSize: image.Point{X: 1075, Y: 605},
		},
	}
}

// GetCompositeLayout returns the composite layer layout
// Matches lines 247-263 of the bash script
func (m *Manager) GetCompositeLayout() []CompositeLayer {
	return []CompositeLayer{
		{ImagePath: filepath.Join(m.AssetsDir, "background_s.jpg"), Position: image.Point{X: 0, Y: 0}},
		{ImagePath: filepath.Join(m.AssetsDir, "weather_gov_hourly_forecast_s.jpg"), Position: image.Point{X: 20, Y: 1130}},
		{ImagePath: filepath.Join(m.AssetsDir, "weather_gov_extended_forecast_s.jpg"), Position: image.Point{X: 2680, Y: 1810}},
		{ImagePath: filepath.Join(m.AssetsDir, "nwac_avalanche_forcast_s.jpg"), Position: image.Point{X: 3420, Y: 420}},
		{ImagePath: filepath.Join(m.AssetsDir, "nwac_stevens_observations_s.jpg"), Position: image.Point{X: 20, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_us2_skykomish.jpg"), Position: image.Point{X: 900, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_w_stevens.jpg"), Position: image.Point{X: 1250, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_big_windy.jpg"), Position: image.Point{X: 1600, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass_b.jpg"), Position: image.Point{X: 1950, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_e_stevens_summit.jpg"), Position: image.Point{X: 2360, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "stevenspassjupiter_s.jpg"), Position: image.Point{X: 905, Y: 285}},
		{ImagePath: filepath.Join(m.AssetsDir, "stevenspassskyline_s.jpg"), Position: image.Point{X: 905, Y: 920}},
		{ImagePath: filepath.Join(m.AssetsDir, "stevenspassschool_s.jpg"), Position: image.Point{X: 905, Y: 1555}},
		{ImagePath: filepath.Join(m.AssetsDir, "stevenspasssnowstake_s.jpg"), Position: image.Point{X: 2010, Y: 285}},
		{ImagePath: filepath.Join(m.AssetsDir, "stevenspasscourtyard_s.jpg"), Position: image.Point{X: 2010, Y: 697}},
		{ImagePath: filepath.Join(m.AssetsDir, "pass_conditions.png"), Position: image.Point{X: 3050, Y: 420}},
		{ImagePath: filepath.Join(m.AssetsDir, "nwac_stevens_avalanche_forcast.png"), Position: image.Point{X: 3100, Y: 60}},
	}
}

// GetPassConditionsImagePath returns the path for the pass conditions overlay
func (m *Manager) GetPassConditionsImagePath() string {
	return filepath.Join(m.AssetsDir, "pass_conditions.png")
}

// GetPassStatusGraphicPath returns the path to the graphic based on pass status
// Returns the appropriate graphic file based on east/west closure status:
// - hw2_open.png = not closed
// - hw2_closed.png = both directions closed
// - hw2_closed_w.png = only west closed
// - hw2_closed_e.png = only east closed
func (m *Manager) GetPassStatusGraphicPath(eastClosed, westClosed bool) string {
	var graphicName string

	if eastClosed && westClosed {
		graphicName = "hw2_closed.png"
	} else if eastClosed {
		graphicName = "hw2_closed_e.png"
	} else if westClosed {
		graphicName = "hw2_closed_w.png"
	} else {
		graphicName = "hw2_open.png"
	}

	return filepath.Join(m.GraphicsDir, graphicName)
}
