# Jasminer X4 Exporter

Export Jasminer X4 dashboard statistics to prometheus

If you have any question do not hesitate to contact me!


## Usage

```sh
git clone https://github.com/jacqueslorentz/jasminer-exporter
cd jasminer-exporter
# Change the --jasminer-uri arguments as you want
docker compose up
```

Command line options:

- `--jasminer-uri`: Jasminer X4 dashboard endpoint (required argument)
- `--listen-address`: address an port the listener will use (default: `:5896`)
- `--telemetry-path`: path on which the exporter metrics will be exposed (default: `/metrics`)
- `--auth-username`: Jasminer X4 dashboard authentication username (default: `root`)
- `--auth-password`: Jasminer X4 dashboard authentication password (default: `root`)

## Further work

Currently to monitor multiple Jasminer X4, you have to deploy multiple instance of this application. Maybe in further commit I will enable multiple endpoints in options.
