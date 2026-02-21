import chalk from "chalk";
import cliProgress from "cli-progress";

/**
 * Source: New helper for CLI progress bars.
 */
export class DownloadProgress {
  private readonly bar: cliProgress.SingleBar;
  private started = false;

  constructor(label: string) {
    this.bar = new cliProgress.SingleBar(
      {
        format:
          `${chalk.blueBright("{label}")} ` +
          `${chalk.cyan("{bar}")} ` +
          `${chalk.white("{percentage}%")} | ` +
          `${chalk.green("{value}/{total}")}`,
        barCompleteChar: "█",
        barIncompleteChar: "░",
        hideCursor: true,
      },
      cliProgress.Presets.shades_classic,
    );
    this.label = label;
  }

  private readonly label: string;

  update(done: number, total: number): void {
    if (!this.started) {
      this.bar.start(total, done, { label: this.label });
      this.started = true;
      return;
    }
    this.bar.setTotal(total);
    this.bar.update(done, { label: this.label });
  }

  stop(): void {
    if (this.started) {
      this.bar.stop();
    }
  }
}
