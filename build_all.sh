mkdir -p binaries

for GOOS in linux darwin windows; do
  for GOARCH in amd64 arm64; do
    OUT="binaries/operator_wrapper_${GOOS}_${GOARCH}"
    [[ $GOOS == windows ]] && OUT="${OUT}.exe"

    echo "🔨 Building $OUT ..."
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 \
      go build -trimpath -ldflags="-s -w" -o "$OUT" main.go
  done
done

echo "All binaries built in ./binaries/"
