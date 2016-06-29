BASEDIR="$( dirname "$0" )"
cd "$BASEDIR"

# all changes will be preserved

g++ datagen.cc -o datagen
g++ judge.cc -o judge
