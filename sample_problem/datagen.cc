#include <cstdio>

int main(int argc, char **argv) {
    freopen("data.in", "w", stdout);
    printf("Username: %s\nTestcaseID: %s\n", argv[1], argv[2]);
    fclose(stdout);
}
