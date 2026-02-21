import chalk from "chalk";

/**
 * Source: New helper for CLI logging style.
 */
export class Logger {
  info(message: string): void {
    this.print(chalk.bgBlue.black(" INFO "), chalk.cyan(message));
  }

  success(message: string): void {
    this.print(chalk.bgGreen.black(" OK "), chalk.greenBright(message));
  }

  warn(message: string): void {
    this.print(chalk.bgYellow.black(" WARN "), chalk.yellowBright(message));
  }

  error(message: string): void {
    this.print(chalk.bgRed.white(" ERROR "), chalk.redBright(message), true);
  }

  failure(message: string): void {
    this.print(
      chalk.bgMagenta.white(" FAIL "),
      chalk.magentaBright(message),
      true,
    );
  }

  private print(label: string, content: string, toError = false): void {
    const ts = chalk.gray(`[${new Date().toISOString()}]`);
    const line = `${ts} ${label} ${content}`;
    if (toError) {
      console.error(line);
    } else {
      console.log(line);
    }
  }
}
