#include <ctime>
#include <cstdlib>
#include <string>
#include <fstream>

// save int score at `score.txt`
// save messages at `message.txt`

// argv[1]: username
// argv[2]: data_id
// argv[3]: input_file_path
// argv[4]: answer_file_path
int main(int argc, char **argv) {
    srand(time(NULL));

    std::ofstream fmsg("message.txt");
    std::ofstream fscore("score.txt");
    std::ifstream fin(argv[3]);
    std::ifstream fans(argv[4]);

    int score = rand() % 20;
    fscore << score << std::endl;

    std::string line;
    fmsg << "message demo" << std::endl;
    fmsg << "score = " << score << std::endl;
    fmsg << "====== input ======" << std::endl;
    while (fin >> line) fmsg << line << std::endl;
    fmsg << "====== answer ======" << std::endl;
    while (fans >> line) fmsg << line << std::endl;
}
