#!/bin/bash

TAG=`git describe --tag --abbrev=0`

platforms=( darwin linux freebsd windows openbsd )

programs=( crabby )

mkdir -p binaries

for prog in "${programs[@]}"
do
  PROG_WITH_TAG=${prog}-${TAG}
  echo "--> Building ${prog}"
  for plat in "${platforms[@]}"
  do
    echo "----> Building for ${plat}/amd64"
    if [ "$plat" = "windows" ]; then
      GOOS=$plat GOARCH=amd64 go build -o ${PROG_WITH_TAG}-win64.exe
      echo "Compressing..."
      zip -9 ${PROG_WITH_TAG}-win64.zip ${PROG_WITH_TAG}-win64.exe
      mv ${PROG_WITH_TAG}-win64.zip binaries/
      rm ${PROG_WITH_TAG}-win64.exe
    else
       OUT="${PROG_WITH_TAG}-${plat}-amd64"
       GOOS=$plat GOARCH=amd64 go build -o $OUT
       echo "Compressing..."
       gzip -f $OUT
       mv ${OUT}.gz binaries/
    fi
  done

  # Build Linux/ARM
  echo "----> Building for linux/arm"
  OUT="${PROG_WITH_TAG}-linux-arm"
  GOOS=linux GOARCH=arm go build -o $OUT
  echo "Compressing..."
  gzip -f $OUT
  mv ${OUT}.gz binaries/
done
