package options

type (
	Options struct {
		Indexer IndexerOpt
		ES      EsOpt
	}
)

func Init() (opt *Options, err error) {
	indexer, err := Indexer()
	if err != nil {
		return
	}

	es, err := ES()
	if err != nil {
		return
	}

	return &Options{
		Indexer: *indexer,
		ES:      *es,
	}, nil
}
