package test_utils

//func WaitForServerReady(t *testing.T, addr string, timeout time.Duration) {
//	t.Helper()
//	deadline := time.Now().Add(timeout)
//	for time.Now().Before(deadline) {
//		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
//		if err == nil {
//			conn.Close()
//			return
//		}
//		time.Sleep(50 * time.Millisecond)
//	}
//	t.Fatalf("Server at %s did not become ready within %v", addr, timeout)
//}
