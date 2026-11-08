{
  writeShellApplication,
  ollama,
  curl,
}:

writeShellApplication {
  name = "ollama-serve";
  runtimeInputs = [ ollama curl ];
  text = ''
    host="''${OLLAMA_HOST:-127.0.0.1:11434}"
    case "$host" in
      http://* | https://*) url="$host" ;;
      *) url="http://$host" ;;
    esac
    export OLLAMA_HOST="$host"

    if [ "$#" -eq 0 ]; then
      set -- qwen3-embedding:0.6b qwen3:1.7b
    fi

    server_pid=""
    if curl -fsS "$url/api/version" >/dev/null 2>&1; then
      echo "Using running Ollama server at $url"
    else
      echo "Starting Ollama server at $url"
      ollama serve &
      server_pid=$!
      trap 'kill "$server_pid" 2>/dev/null || true' EXIT INT TERM
      until curl -fsS "$url/api/version" >/dev/null 2>&1; do
        if ! kill -0 "$server_pid" 2>/dev/null; then
          echo "Ollama server exited" >&2
          exit 1
        fi
        sleep 0.5
      done
    fi

    for model in "$@"; do
      ollama pull "$model"
    done

    if [ -n "$server_pid" ]; then
      echo "Ollama server running, press Ctrl-C to stop"
      wait "$server_pid"
    fi
  '';
}
