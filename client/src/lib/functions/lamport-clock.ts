class LamportClock {
  private counter: number;

  constructor() {
      this.counter = 0;
  }

  // Increment the clock on internal events
  public tick(): number {
      this.counter++;
      return this.counter;
  }

  // Update the clock on receiving a message
  public receive(timestamp: number): number {
      this.counter = Math.max(this.counter, timestamp) + 1;
      return this.counter;
  }

  // Get the current time
  public getTime(): number {
      return this.counter;
  }
}

export default LamportClock;