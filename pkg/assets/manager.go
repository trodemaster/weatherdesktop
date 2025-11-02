package assets

import (
	"image"
	"path/filepath"
)

// Asset represents a downloadable or scrapeable asset
type Asset struct {
	Name          string
	URL           string
	LocalPath     string
	ProcessedPath string
	CropParams    *CropParams
	ResizeParams  *ResizeParams
}

// CropParams defines crop coordinates
type CropParams struct {
	Width  int
	Height int
	X      int
	Y      int
}

// ResizeParams defines resize parameters
type ResizeParams struct {
	Width   int
	Height  int
	Percent int // If set, use percentage instead of absolute size
}

// CompositeLayer defines where to place an image in the final composite
type CompositeLayer struct {
	ImagePath string
	Position  image.Point
}

// Manager handles all asset configuration
type Manager struct {
	BaseDir    string
	AssetsDir  string
	RenderedDir string
}

// NewManager creates a new asset manager
func NewManager(baseDir string) *Manager {
	return &Manager{
		BaseDir:     baseDir,
		AssetsDir:   filepath.Join(baseDir, "assets"),
		RenderedDir: filepath.Join(baseDir, "rendered"),
	}
}

// GetDownloadAssets returns all assets that need to be downloaded via HTTP
func (m *Manager) GetDownloadAssets() []Asset {
	return []Asset{
		{
			Name:      "GOES18 North Pacific",
			URL:       "https://cdn.star.nesdis.noaa.gov/GOES18/ABI/SECTOR/np/GEOCOLOR/latest.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "GOES18_north_pacific.jpg"),
		},
		{
			Name:      "WSDOT Stevens Pass",
			URL:       "https://images.wsdot.wa.gov/nc/002vc06430.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass.jpg"),
		},
		{
			Name:      "WSDOT US2 Skykomish",
			URL:       "https://images.wsdot.wa.gov/nw/002vc04558.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "wsdot_us2_skykomish.jpg"),
		},
		{
			Name:      "WSDOT E Stevens Summit",
			URL:       "https://images.wsdot.wa.gov/nc/002vc06458.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "wsdot_e_stevens_summit.jpg"),
		},
		{
			Name:      "WSDOT Big Windy",
			URL:       "https://images.wsdot.wa.gov/nc/002vc06300.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "wsdot_big_windy.jpg"),
		},
		{
			Name:      "WSDOT W Stevens",
			URL:       "https://images.wsdot.wa.gov/nc/002vc06190.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "wsdot_w_stevens.jpg"),
		},
		{
			Name:      "Stevens Pass Courtyard",
			URL:       "https://streamer8.brownrice.com/cam-images/stevenspasscourtyard.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "stevenspasscourtyard.jpg"),
		},
		{
			Name:      "Stevens Pass Snow Stake",
			URL:       "https://streamer8.brownrice.com/cam-images/stevenspasssnowstake.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "stevenspasssnowstake.jpg"),
		},
		{
			Name:      "Stevens Pass Jupiter",
			URL:       "https://streamer8.brownrice.com/cam-images/stevenspassjupiter.jpg",
			LocalPath: filepath.Join(m.AssetsDir, "stevenspassjupiter.jpg"),
		},
	}
}

// ScrapeTarget represents a website to scrape
type ScrapeTarget struct {
	Name       string
	URL        string
	Selector   string
	OutputPath string
	WaitTime   int // milliseconds
}

// GetScrapeTargets returns all websites that need to be scraped
func (m *Manager) GetScrapeTargets() []ScrapeTarget {
	return []ScrapeTarget{
		{
			Name:       "Weather.gov Hourly Forecast",
			URL:        "https://forecast.weather.gov/MapClick.php?lat=47.7456&lon=-121.0892&unit=0&lg=english&FcstType=graphical",
			Selector:   "img[src*=\"meteograms/Plotter.php\"]", // Just the meteogram graph panels
			OutputPath: filepath.Join(m.AssetsDir, "weather_gov_hourly_forecast.png"),
			WaitTime:   5000, // Page loads slowly, meteogram is dynamically generated
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
			WaitTime:   5000,
		},
		{
			Name:       "NWAC Avalanche Forecast",
			URL:        "https://nwac.us/avalanche-forecast/#/stevens-pass",
			Selector:   "#nac-tab-resizer > div > div:nth-child(1) > div > div.nac-danger.nac-mb-4 > div.nac-row > div.nac-dangerToday.nac-col-lg-8.nac-mb-3 > div.nac-dangerGraphic",
			OutputPath: filepath.Join(m.AssetsDir, "nwac_stevens_avalanche_forcast.png"),
			WaitTime:   1000,
		},
		{
			Name:       "NWAC Avalanche Forecast Map",
			URL:        "https://nwac.us",
			Selector:   "#danger-map-widget",
			OutputPath: filepath.Join(m.AssetsDir, "nwac_avalanche_forcast.png"),
			WaitTime:   1000,
		},
	}
}

// GetWSDOTHTMLTarget returns the WSDOT pass status HTML extraction target
func (m *Manager) GetWSDOTHTMLTarget() ScrapeTarget {
	return ScrapeTarget{
		Name:       "WSDOT Stevens Pass Status",
		URL:        "https://wsdot.com/travel/real-time/mountainpasses/stevens",
		Selector:   "#index > div:nth-child(7) > div.full-width.column-container.mountain-pass > div.column-1",
		OutputPath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass.html"),
		WaitTime:   1000,
	}
}

// GetCropAssets returns all assets that need cropping and resizing
func (m *Manager) GetCropAssets() []Asset {
	return []Asset{
		{
			Name:      "Background Satellite",
			LocalPath: filepath.Join(m.AssetsDir, "GOES18_north_pacific.jpg"),
			ProcessedPath: filepath.Join(m.AssetsDir, "background_s.jpg"),
			CropParams: &CropParams{Width: 7200, Height: 4050, X: 0, Y: 0},
			ResizeParams: &ResizeParams{Width: 3840, Height: 0}, // Maintain aspect ratio
		},
		{
			Name:      "NWAC Avalanche Forecast Map",
			LocalPath: filepath.Join(m.AssetsDir, "nwac_avalanche_forcast.png"),
			ProcessedPath: filepath.Join(m.AssetsDir, "nwac_avalanche_forcast_s.jpg"),
			CropParams: &CropParams{Width: 400, Height: 520, X: 65, Y: 110},
		},
		{
			Name:      "NWAC Stevens Avalanche Forecast",
			LocalPath: filepath.Join(m.AssetsDir, "nwac_stevens_avalanche_forcast.png"),
			ProcessedPath: filepath.Join(m.AssetsDir, "nwac_stevens_avalanche_forcast_s.jpg"),
			CropParams: &CropParams{Width: 1086, Height: 380, X: 0, Y: 25},
		},
		{
			Name:      "NWAC Stevens Observations",
			LocalPath: filepath.Join(m.AssetsDir, "nwac_stevens_observations.png"),
			ProcessedPath: filepath.Join(m.AssetsDir, "nwac_stevens_observations_s.jpg"),
			CropParams: &CropParams{Width: 1140, Height: 1439, X: 0, Y: 0},
			ResizeParams: &ResizeParams{Percent: 75},
		},
		{
			Name:      "Stevens Pass Courtyard",
			LocalPath: filepath.Join(m.AssetsDir, "stevenspasscourtyard.jpg"),
			ProcessedPath: filepath.Join(m.AssetsDir, "stevenspasscourtyard_s.jpg"),
			CropParams: &CropParams{Width: 1920, Height: 1080, X: 0, Y: 0},
			ResizeParams: &ResizeParams{Percent: 50},
		},
		{
			Name:      "Stevens Pass Snow Stake",
			LocalPath: filepath.Join(m.AssetsDir, "stevenspasssnowstake.jpg"),
			ProcessedPath: filepath.Join(m.AssetsDir, "stevenspasssnowstake_s.jpg"),
			CropParams: &CropParams{Width: 1920, Height: 1080, X: 0, Y: 0},
			ResizeParams: &ResizeParams{Percent: 50},
		},
		{
			Name:      "Weather.gov Extended Forecast",
			LocalPath: filepath.Join(m.AssetsDir, "weather_gov_extended_forecast.png"),
			ProcessedPath: filepath.Join(m.AssetsDir, "weather_gov_extended_forecast_s.jpg"),
			CropParams: &CropParams{Width: 1146, Height: 300, X: 0, Y: 100},
		},
		{
			Name:      "WSDOT Stevens Pass (Big)",
			LocalPath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass.jpg"),
			ProcessedPath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass_b.jpg"),
			ResizeParams: &ResizeParams{Percent: 119},
		},
	}
}

// GetCompositeLayout returns the layer positions for the final composite
// Matches lines 247-263 of the bash script
func (m *Manager) GetCompositeLayout() []CompositeLayer {
	return []CompositeLayer{
		{ImagePath: filepath.Join(m.AssetsDir, "background_s.jpg"), Position: image.Point{X: 0, Y: 0}},
		{ImagePath: filepath.Join(m.AssetsDir, "weather_gov_hourly_forecast.png"), Position: image.Point{X: 20, Y: 1130}},
		{ImagePath: filepath.Join(m.AssetsDir, "weather_gov_extended_forecast_s.jpg"), Position: image.Point{X: 2680, Y: 1860}},
		{ImagePath: filepath.Join(m.AssetsDir, "nwac_avalanche_forcast_s.jpg"), Position: image.Point{X: 3420, Y: 420}},
		{ImagePath: filepath.Join(m.AssetsDir, "nwac_stevens_observations_s.jpg"), Position: image.Point{X: 20, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_us2_skykomish.jpg"), Position: image.Point{X: 900, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_w_stevens.jpg"), Position: image.Point{X: 1250, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_big_windy.jpg"), Position: image.Point{X: 1600, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_stevens_pass_b.jpg"), Position: image.Point{X: 1950, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "wsdot_e_stevens_summit.jpg"), Position: image.Point{X: 2360, Y: 20}},
		{ImagePath: filepath.Join(m.AssetsDir, "stevenspassjupiter.jpg"), Position: image.Point{X: 900, Y: 285}},
		{ImagePath: filepath.Join(m.AssetsDir, "stevenspasssnowstake_s.jpg"), Position: image.Point{X: 910, Y: 1730}},
		{ImagePath: filepath.Join(m.AssetsDir, "stevenspasscourtyard_s.jpg"), Position: image.Point{X: 1600, Y: 1730}},
		{ImagePath: filepath.Join(m.AssetsDir, "pass_conditions.png"), Position: image.Point{X: 3150, Y: 420}},
		{ImagePath: filepath.Join(m.AssetsDir, "nwac_stevens_avalanche_forcast_s.jpg"), Position: image.Point{X: 3100, Y: 40}},
	}
}

// GetPassConditionsImagePath returns the path for the pass conditions overlay
func (m *Manager) GetPassConditionsImagePath() string {
	return filepath.Join(m.AssetsDir, "pass_conditions.png")
}

