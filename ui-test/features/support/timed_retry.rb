def timed_retry(options = {})
  retries = options.fetch(:retries, 10)
  sleep_time = options.fetch(:sleep, 0.2)
  begin
    yield
  rescue Exception => e
    retries -= 1
    if retries > 0
      sleep sleep_time
      retry
    end
    raise e
  end
end