BASEDIR="$( dirname "$0" )"
cd "$BASEDIR"

./datagen $1 $2 > data.in
