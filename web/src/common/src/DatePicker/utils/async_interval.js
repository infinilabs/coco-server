export class AsyncInterval {
  timeoutId = null;
  isStopped = false;
  __pendingFn = () => {};

  constructor(fn, refreshInterval) {
    this.setAsyncInterval(fn, refreshInterval);
  }

  setAsyncInterval = (fn, milliseconds) => {
    if (!this.isStopped) {
      this.timeoutId = window.setTimeout(async () => {
        this.__pendingFn = await fn();
        this.setAsyncInterval(fn, milliseconds);
      }, milliseconds);
    }
  };

  stop = () => {
    this.isStopped = true;
    if (this.timeoutId !== null) {
      window.clearTimeout(this.timeoutId);
    }
  };
}
