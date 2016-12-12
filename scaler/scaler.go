package scaler

type Scaler struct {
	Config *Config
}

func NewScaler(c *Config) *Scaler {
	return &Scaler{
		Config: c,
	}
}

func (s *Scaler) Run() {

}
