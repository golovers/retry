# retry
retry is a simple http-retry client that support backoff policy such as exponential backoff with max retries...

```
# retry
go get github.com/golovers/retry

# dependencies
go get github.com/cenkalti/backoff
go get github.com/sirupsen/logrus
```

## Usage

```go
c := retry.New()
req, _ := http.NewRequest(http.MethodGet, "https://www.github.com/pthethanh", nil)

// Using default backoff policy and default retry func
rs, err := c.Do(req)
logrus.Infof("response: %+v, err: %v", rs, err)

// Using custom backoff policy
rs, err = c.DoWithBackOff(req, backoff.NewExponentialBackOff())
logrus.Infof("response: %+v, err: %v", rs, err)

// Using custom backoff policy and custom retry func
rs, err = c.DoWithRetryFunc(req, backoff.NewConstantBackOff(1*time.Second), func(rs *http.Response) bool {
    return rs.StatusCode == http.StatusInternalServerError
})
logrus.Infof("response: %+v, err: %v", rs, err)
```
