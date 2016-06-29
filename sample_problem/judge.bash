BASEDIR="$( dirname "$0" )"
cd "$BASEDIR"

# save int score at `score.txt`
# save messages at `message.txt`
#
# $1: username
# $2: data_id
# $3: input_file_path
# $4: answer_file_path

./judge $1 $2 $3 $4
