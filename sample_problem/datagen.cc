#include <cstdio>

// save data at `data.in`
//
// argv[1]: username
// argv[2]: data_id
int main(int argc, char **argv) {
    freopen("data.in", "w", stdout);
    printf("Username: %s\nTestcaseID: %s\n", argv[1], argv[2]);
    fclose(stdout);
}
