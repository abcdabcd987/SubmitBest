BASEDIR="$( dirname "$0" )"
cd "$BASEDIR"

# save data at `data.in`
#
# $1: username
# $2: data_id

./datagen $1 $2
#mo
