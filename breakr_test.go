package breakr

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/xh3b4sd/tracer"
)

func Test_Breakr_Default(t *testing.T) {
	var testError = &tracer.Error{
		Kind: "testError",
	}

	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		val uint
		mat func(err error) bool
	}{
		// case 0
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() == 2 {
					return Cancel
				}

				return fmt.Errorf("test error")
			},
			val: 2,
			mat: func(err error) bool {
				return errors.Is(err, Cancel)
			},
		},
		// case 1
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() == 2 {
					return Repeat
				}
				if cou.Cou() == 3 {
					return nil
				}
				return fmt.Errorf("test error")
			},
			val: 3,
			mat: func(err error) bool {
				return errors.Is(err, nil)
			},
		},
		// case 2
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() == 4 {
					return tracer.Mask(Cancel)
				}

				return testError
			},
			val: 3,
			mat: func(err error) bool {
				return errors.Is(err, testError)
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			var b Interface
			{
				b = Default()
			}

			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			cou.Res()

			err := b.Execute(tc.act)
			if !tc.mat(err) {
				t.Fatalf("expected error matcher to match")
			}

			if cou.Cou() != tc.val {
				t.Fatalf("expected %#v got %#v", tc.val, cou.Cou())
			}
		})
	}
}

func Test_Breakr_Constant_Cancel_Repeat(t *testing.T) {
	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		fai uint
		cou uint
		mat func(err error) bool
	}{
		// case 0
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() == 2 {
					return Cancel
				}

				return fmt.Errorf("test error")
			},
			fai: 3,
			cou: 2,
			mat: func(err error) bool {
				return errors.Is(err, Cancel)
			},
		},
		// case 1
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() == 2 {
					return tracer.Mask(Repeat)
				}
				if cou.Cou() == 3 {
					return nil
				}

				return fmt.Errorf("test error")
			},
			fai: 3,
			cou: 3,
			mat: func(err error) bool {
				return errors.Is(err, nil)
			},
		},
		// case 2
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() == 4 {
					return tracer.Mask(Cancel)
				}

				return fmt.Errorf("test error")
			},
			fai: 8,
			cou: 4,
			mat: func(err error) bool {
				return errors.Is(err, Cancel)
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			var err error

			var b Interface
			{
				b = New(Config{
					Failure: Failure{
						Budget: tc.fai,
						Cooler: -1,
					},
					Timeout: Timeout{
						Action: -1,
					},
				})
			}

			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			cou.Res()

			err = b.Execute(tc.act)
			if !tc.mat(err) {
				t.Fatalf("expected error matcher to match")
			}

			if cou.Cou() != tc.cou {
				t.Fatalf("expected %#v got %#v", tc.cou, cou.Cou())
			}
		})
	}
}

func Test_Breakr_Constant_Errors(t *testing.T) {
	var testError = &tracer.Error{
		Kind: "testError",
	}

	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
	}{
		// Case 0 tests error handling.
		{
			act: func() error {
				cou.Inc()
				return testError
			},
		},
		// Case 1 tests error handling with masking.
		{
			act: func() error {
				cou.Inc()
				return tracer.Mask(testError)
			},
		},
	}

	var err error

	// Note that the budget implementation is reused across all test cases in
	// order to ensure the reusability of a single budget instance. This is a
	// feature we want to ensure. Using up the configured budget of a given
	// budget instance should only happen in isolation and not affect
	// consecutive calls of the same instance.
	var b Interface
	{
		b = New(Config{
			Failure: Failure{
				Budget: 5,
				Cooler: -1,
			},
			Timeout: Timeout{
				Action: -1,
			},
		})
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			cou.Res()

			err = b.Execute(tc.act)
			if !errors.Is(err, testError) {
				t.Fatalf("expected test error")
			}

			// We expect five executions because we hard coded the errors
			// returned while the budget is five. That means we should see five
			// attempts of executing the operation.
			if cou.Cou() != 5 {
				t.Fatalf("expected %#v got %#v", 5, cou.Cou())
			}
		})
	}
}

func Test_Breakr_Constant_Retries(t *testing.T) {
	var testError = &tracer.Error{
		Kind: "testError",
	}

	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		fai uint
		val uint
	}{
		// Case 0 tests a budget of one which should result in one execution.
		{
			act: func() error {
				cou.Inc()
				return nil
			},
			fai: 1,
			val: 1,
		},
		// Case 1 tests a budget of two which should result in one execution.
		{
			act: func() error {
				cou.Inc()
				return nil
			},
			fai: 2,
			val: 1,
		},
		// Case 3 tests a budget of nine which should result in one execution.
		{
			act: func() error {
				cou.Inc()
				return nil
			},
			fai: 9,
			val: 1,
		},
		// Case 4 tests a budget of one which should result in one execution due
		// to the returned error.
		{
			act: func() error {
				cou.Inc()
				return testError
			},
			fai: 1,
			val: 1,
		},
		// Case 5 tests a budget of two which should result in two executions
		// due to the returned error.
		{
			act: func() error {
				cou.Inc()
				return testError
			},
			fai: 2,
			val: 2,
		},
		// Case 6 tests a budget of nine which should result in nine executions
		// due to the returned error.
		{
			act: func() error {
				cou.Inc()
				return tracer.Mask(testError)
			},
			fai: 9,
			val: 9,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			var err error

			var b Interface
			{
				b = New(Config{
					Failure: Failure{
						Budget: tc.fai,
						Cooler: -1,
					},
					Timeout: Timeout{
						Action: -1,
					},
				})
			}

			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			cou.Res()

			err = b.Execute(tc.act)
			if errors.Is(err, testError) {
				// Simply fall through in case we get the test error because in
				// case we get it we produced it purposefully in order to check
				// the exection results in any given situation.
			} else if err != nil {
				t.Fatal(err)
			}

			if cou.Cou() != tc.val {
				t.Fatalf("expected %#v got %#v", tc.val, cou.Cou())
			}
		})
	}
}

func Test_Breakr_Limiter_Cou(t *testing.T) {
	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		bud uint
		cou uint
		coo time.Duration
	}{
		// case 0
		{
			act: func() error { cou.Inc(); time.Sleep(0 * time.Millisecond); return nil },
			bud: 1,
			cou: 9,
			coo: 100 * time.Millisecond,
		},
		// case 1
		{
			act: func() error { cou.Inc(); time.Sleep(1 * time.Millisecond); return nil },
			bud: 2,
			cou: 6,
			coo: 300 * time.Millisecond,
		},
		// case 2
		{
			act: func() error { cou.Inc(); time.Sleep(0 * time.Millisecond); return nil },
			bud: 3,
			cou: 15,
			coo: 200 * time.Millisecond,
		},
		// case 3
		{
			act: func() error { cou.Inc(); time.Sleep(5 * time.Millisecond); return nil },
			bud: 5,
			cou: 10,
			coo: 500 * time.Millisecond,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			var b Interface
			{
				b = New(Config{
					Limiter: Limiter{
						Budget: tc.bud,
						Cooler: tc.coo,
					},
				})
			}

			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			cou.Res()

			var wai sync.WaitGroup

			for i := 0; i < 9; i++ {
				for i := 0; i < 9; i++ {
					wai.Add(1)

					go func(act func() error) {
						defer wai.Done()

						err := b.Execute(act)
						if IsFilled(err) {
							// fall through
						} else if err != nil {
							panic(err)
						}
					}(tc.act)

					time.Sleep(10 * time.Millisecond)
				}
			}

			wai.Wait()

			if !bet(cou.Cou(), tc.cou) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.cou, cou.Cou()))
			}
		})
	}
}

func Test_Breakr_Limiter_Max(t *testing.T) {
	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		bud uint
		max uint
		coo time.Duration
	}{
		// case 0
		{
			act: func() error { cou.Inc(); defer cou.Dec(); time.Sleep(200 * time.Millisecond); return nil },
			bud: 1,
			max: 1,
			coo: 100 * time.Millisecond,
		},
		// case 1
		{
			act: func() error { cou.Inc(); defer cou.Dec(); time.Sleep(100 * time.Millisecond); return nil },
			bud: 2,
			max: 2,
			coo: 300 * time.Millisecond,
		},
		// case 2
		{
			act: func() error { cou.Inc(); defer cou.Dec(); time.Sleep(50 * time.Millisecond); return nil },
			bud: 3,
			max: 3,
			coo: 20 * time.Millisecond,
		},
		// case 3
		{
			act: func() error { cou.Inc(); defer cou.Dec(); time.Sleep(50 * time.Millisecond); return nil },
			bud: 5,
			max: 5,
			coo: 500 * time.Millisecond,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			var b Interface
			{
				b = New(Config{
					Limiter: Limiter{
						Budget: tc.bud,
						Cooler: tc.coo,
					},
				})
			}

			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			{
				cou.Res()
			}

			var wai sync.WaitGroup

			for i := 0; i < 9; i++ {
				for i := 0; i < 9; i++ {
					wai.Add(1)

					go func(act func() error) {
						defer wai.Done()

						err := b.Execute(act)
						if IsFilled(err) {
							// fall through
						} else if err != nil {
							panic(err)
						}
					}(tc.act)

					time.Sleep(10 * time.Millisecond)
				}
			}

			wai.Wait()

			if cou.Max() != tc.max {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.max, cou.Max()))
			}
		})
	}
}

func Test_Breakr_Limiter_Cooler(t *testing.T) {
	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		bud uint
		coo time.Duration
	}{
		// case 0
		{
			bud: 1,
			coo: 1 * time.Second,
		},
		// case 1
		{
			bud: 1,
			coo: 2 * time.Second,
		},
		// case 2
		{
			bud: 2,
			coo: 1 * time.Second,
		},
		// case 3
		{
			bud: 2,
			coo: 2 * time.Second,
		},
		// case 4
		{
			bud: 3,
			coo: 1 * time.Second,
		},
		// case 5
		{
			bud: 3,
			coo: 5 * time.Second,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			var b Interface
			{
				b = New(Config{
					Limiter: Limiter{
						Budget: tc.bud,
						Cooler: tc.coo,
					},
				})
			}

			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			{
				cou.Res()
			}

			// Ensure that the first action can be executed without delay while the
			// queue is empty.
			{
				for i := 0; i < int(tc.bud); i++ {
					var sta time.Time
					{
						sta = time.Now().UTC()
					}

					{
						err := b.Execute(func() error { cou.Inc(); return nil })
						if err != nil {
							t.Fatal(err)
						}
					}

					{
						if time.Since(sta) > 10*time.Millisecond {
							t.Fatal("limiter must not delay action execution")
						}
						if cou.Cou() != uint(i+1) {
							t.Fatalf("action must be executed %d time(s)", i+1)
						}
					}
				}
			}

			{
				for i := 0; i < 10; i++ {
					var sta time.Time
					{
						sta = time.Now().UTC()
					}

					{
						err := b.Execute(func() error { cou.Inc(); return nil })
						if !IsFilled(err) {
							t.Fatal(err)
						}
					}

					{
						if time.Since(sta) > 10*time.Millisecond {
							t.Fatal("limiter must not delay action execution")
						}
						if cou.Cou() != tc.bud {
							t.Fatalf("action must be executed %d time(s)", tc.bud)
						}
					}

					{
						time.Sleep(tc.coo / 20)
					}
				}
			}

			// The test is configured to allow N executions within T seconds.
			// The throttled actions above took T/2 seconds to execute in total.
			// We wait another T/2 seconds and ensure that the configured action
			// can be executed without delay again, once the limiter queue to
			// clear.
			{
				time.Sleep(tc.coo / 2)
			}

			{
				for i := 0; i < int(tc.bud); i++ {
					var sta time.Time
					{
						sta = time.Now().UTC()
					}

					{
						err := b.Execute(func() error { cou.Inc(); return nil })
						if err != nil {
							t.Fatal(err)
						}
					}

					{
						if time.Since(sta) > 10*time.Millisecond {
							t.Fatal("limiter must not delay action execution")
						}
						if cou.Cou() != tc.bud+uint(i+1) {
							t.Fatalf("action must be executed %d time(s)", tc.bud+uint(i+1))
						}
					}
				}
			}

			{
				time.Sleep(tc.coo)
			}

			{
				var sta time.Time
				{
					sta = time.Now().UTC()
				}

				{
					err := b.Execute(func() error { cou.Inc(); return nil })
					if err != nil {
						t.Fatal(err)
					}
				}

				{
					if time.Since(sta) > 10*time.Millisecond {
						t.Fatal("limiter must not delay action execution")
					}
					if cou.Cou() != (2*tc.bud)+1 {
						t.Fatalf("action must be executed %d time(s)", (2*tc.bud)+1)
					}
				}
			}
		})
	}
}

func Test_Breakr_Timeout_Close(t *testing.T) {
	var clo chan struct{}
	var exe int

	testCases := []struct {
		act func() error
		exe int
	}{
		// case 0
		{
			act: func() error {
				exe++

				if exe == 2 {
					close(clo)
					return nil
				}

				return nil
			},
			exe: 2,
		},
		// case 1
		{
			act: func() error {
				exe++

				if exe == 4 {
					close(clo)
					return nil
				}

				return nil
			},
			exe: 4,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			clo = make(chan struct{})
			exe = 0

			var err error

			var b Interface
			{
				b = New(Config{
					Failure: Failure{
						Cooler: -1,
					},
					Success: Success{
						Budget: 5,
					},
					Timeout: Timeout{
						Action: 100 * time.Millisecond,
						Closer: clo,
					},
				})
			}

			err = b.Execute(tc.act)
			if !IsClosed(err) {
				t.Fatalf("expected error to match")
			}

			if exe != tc.exe {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.exe, exe))
			}
		})
	}
}

func Test_Breakr_Timeout_Error(t *testing.T) {
	var testError = &tracer.Error{
		Kind: "testError",
	}

	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		val uint
		mat func(err error) bool
	}{
		// case 0
		{
			act: func() error {
				cou.Inc()
				return testError
			},
			val: 1,
			mat: func(err error) bool {
				return errors.Is(err, testError)
			},
		},
		// case 1
		{
			act: func() error {
				cou.Inc()
				return tracer.Mask(testError)
			},
			val: 1,
			mat: func(err error) bool {
				return errors.Is(err, testError)
			},
		},
		// case 2
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() == 3 {
					return testError
				}

				return nil
			},
			val: 3,
			mat: func(err error) bool {
				return errors.Is(err, testError)
			},
		},
		// case 3
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() == 99 {
					return testError
				}

				return nil
			},
			val: 5,
			mat: func(err error) bool {
				return errors.Is(err, nil)
			},
		},
	}

	var err error

	// Note that the budget implementation is reused across all test cases in
	// order to ensure the reusability of a single budget instance. This is a
	// feature we want to ensure. Using up the configured budget of a given
	// budget instance should only happen in isolation and not affect
	// consecutive calls of the same instance.
	var b Interface
	{
		b = New(Config{
			Failure: Failure{
				Budget: 1,
				Cooler: -1,
			},
			Success: Success{
				Budget: 5,
			},
			Timeout: Timeout{
				Action: 100 * time.Millisecond,
			},
		})
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			cou.Res()

			err = b.Execute(tc.act)
			if !tc.mat(err) {
				t.Fatalf("expected error to match")
			}

			if cou.Cou() != tc.val {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.val, cou.Cou()))
			}
		})
	}
}

func Test_Breakr_Timeout_Repeat_1(t *testing.T) {
	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		val uint
		mat func(err error) bool
	}{
		// case 0
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() <= 1 {
					time.Sleep(600 * time.Millisecond)
					return nil
				}

				if cou.Cou() <= 3 {
					return tracer.Mask(Repeat)
				}

				return nil
			},
			val: 4,
			mat: func(err error) bool {
				return errors.Is(err, nil)
			},
		},
		// case 1
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() <= 3 {
					time.Sleep(180 * time.Millisecond)
					return nil
				}

				if cou.Cou() <= 6 {
					return tracer.Mask(Repeat)
				}

				return nil
			},
			val: 2, // nil is returned after 180ms, which is within the 5*100ms tolerance
			mat: func(err error) bool {
				return errors.Is(err, nil) // Repeat is never called because 3*200ms > 5*100ms
			},
		},
		// case 2
		{
			act: func() error {
				cou.Inc()

				time.Sleep(250 * time.Millisecond)

				return tracer.Mask(Cancel)
			},
			val: 3, // the 100ms timeout executes act 3 times within 250ms
			mat: func(err error) bool {
				return errors.Is(err, Cancel) // Cancel is returned after 250ms, which is within the 5*100ms tolerance
			},
		},
	}

	var err error

	// Note that the budget implementation is reused across all test cases in
	// order to ensure the reusability of a single budget instance. This is a
	// feature we want to ensure. Using up the configured budget of a given
	// budget instance should only happen in isolation and not affect
	// consecutive calls of the same instance.
	var b Interface
	{
		b = New(Config{
			Failure: Failure{
				Budget: 5,
				Cooler: -1,
			},
			Limiter: Limiter{
				Budget: 10,
			},
			Timeout: Timeout{
				Action: 100 * time.Millisecond,
				Budget: 5,
			},
		})
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			for i := 0; i < 10; i++ {
				// Note that this counter has to be reset for each test in order to
				// lead to accurate results.
				cou.Res()

				err = b.Execute(tc.act)
				if !tc.mat(err) {
					fmt.Printf("%#v\n", err)
					t.Fatalf("expected error to match")
				}

				if cou.Cou() != tc.val {
					t.Fatalf("\n\n%s\n", cmp.Diff(tc.val, cou.Cou()))
				}
			}
		})
	}
}

func Test_Breakr_Timeout_Repeat_2(t *testing.T) {
	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		val uint
		mat func(err error) bool
	}{
		// case 0
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() <= 3 {
					return tracer.Mask(Repeat)
				}

				return nil
			},
			val: 8,
			mat: func(err error) bool {
				return errors.Is(err, nil)
			},
		},
		// case 1
		{
			act: func() error {
				cou.Inc()

				if cou.Cou() <= 6 {
					return tracer.Mask(Repeat)
				}

				return nil
			},
			val: 11,
			mat: func(err error) bool {
				return errors.Is(err, nil)
			},
		},
	}

	var err error

	// Note that the budget implementation is reused across all test cases in
	// order to ensure the reusability of a single budget instance. This is a
	// feature we want to ensure. Using up the configured budget of a given
	// budget instance should only happen in isolation and not affect
	// consecutive calls of the same instance.
	var b Interface
	{
		b = New(Config{
			Failure: Failure{
				Budget: 1,
				Cooler: -1,
			},
			Success: Success{
				Budget: 5,
			},
			Timeout: Timeout{
				Action: 100 * time.Millisecond,
			},
		})
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			cou.Res()

			err = b.Execute(tc.act)
			if !tc.mat(err) {
				t.Fatalf("expected error to match")
			}

			if cou.Cou() != tc.val {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.val, cou.Cou()))
			}
		})
	}
}

func Test_Breakr_Timeout_Timeout(t *testing.T) {
	var cou *counter
	{
		cou = &counter{}
	}

	testCases := []struct {
		act func() error
		coo time.Duration
		glo time.Duration
		min uint
		mat func(err error) bool
	}{
		// case 0
		{
			act: func() error {
				cou.Inc()
				time.Sleep(10 * time.Millisecond)
				return tracer.Mask(Repeat)
			},
			coo: -1,
			glo: 100 * time.Millisecond,
			min: 9,
			mat: IsPassed,
		},
		// case 0
		{
			act: func() error {
				cou.Inc()
				time.Sleep(500 * time.Millisecond)
				return Repeat
			},
			coo: -1,
			glo: 100 * time.Millisecond,
			min: 1,
			mat: IsPassed,
		},
		// case 2
		{
			act: func() error {
				cou.Inc()

				time.Sleep(10 * time.Millisecond)

				if cou.Cou() <= 2 {
					return Repeat
				}

				return nil // 3. time executed finishes the 1 required execution.
			},
			coo: 100 * time.Millisecond,
			glo: 280 * time.Millisecond, // 2*100 + 3*10 + 1*50 for not closing
			min: 3,
			mat: func(err error) bool {
				return errors.Is(err, nil)
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
			// Note that this counter has to be reset for each test in order to
			// lead to accurate results.
			cou.Res()

			var err error

			var b Interface
			{
				b = New(Config{
					Failure: Failure{
						Budget: 1,
						Cooler: tc.coo,
					},
					Timeout: Timeout{
						Global: tc.glo,
					},
				})
			}

			var sta time.Time
			{
				sta = time.Now()
			}

			err = b.Execute(tc.act)
			if !tc.mat(err) {
				t.Fatalf("expected error to match")
			}

			var tim time.Duration
			{
				tim = time.Since(sta)
			}

			if tim > tc.glo+20*time.Millisecond {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.glo, tim))
			}
			if cou.Cou() < tc.min {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.min, cou.Cou()))
			}
		})
	}
}

type counter struct {
	cou uint
	max uint
	mut sync.Mutex
}

func (c *counter) Cou() uint {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.cou
}

func (c *counter) Dec() {
	{
		c.mut.Lock()
		defer c.mut.Unlock()
	}

	{
		c.cou--
	}
}

func (c *counter) Inc() {
	{
		c.mut.Lock()
		defer c.mut.Unlock()
	}

	{
		c.cou++
		if c.cou > c.max {
			c.max = c.cou
		}
	}
}

func (c *counter) Max() uint {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.max
}

func (c *counter) Res() {
	{
		c.mut.Lock()
		defer c.mut.Unlock()
	}

	{
		c.cou = 0
		c.max = 0
	}
}

func bet(cou uint, mid uint) bool {
	if cou < mid-2 {
		return false
	}
	if cou > mid+2 {
		return false
	}

	return true
}
