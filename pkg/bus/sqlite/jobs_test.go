package sqlite_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/get_jobs"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/bus/sqlite"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/storage"
	sq "github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

func Test_JobsStore(t *testing.T) {
	bus.Marshallable.Register(asyncCommand{}, func(s string) (bus.AsyncRequest, error) {
		return storage.UnmarshalJSON[asyncCommand](s)
	})

	t.Run("should be able to queue multiple jobs", func(t *testing.T) {
		store, _ := prepareDatabase(t)

		err := store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}, asyncCommand{
				Key__:   "key2",
				Topic__: "group2",
			})

		assert.Nil(t, err)

		data, err := store.GetAllJobs(context.Background(), get_jobs.Query{})

		assert.Nil(t, err)
		assert.HasLength(t, 2, data.Data)
	})

	t.Run("should be able to retrieve next pending jobs", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}, asyncCommand{
				Key__:   "key2",
				Topic__: "group2",
			}))

		jobs, err := store.GetNextPendingJobs(context.Background())

		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)
	})

	t.Run("should only retrieve one job per topic", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}, asyncCommand{
				Key__:   "key2",
				Topic__: "group2",
			}, asyncCommand{
				Key__:   "key3",
				Topic__: "group1",
			}))

		jobs, err := store.GetNextPendingJobs(context.Background())

		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)
		assert.Equal(t, "key1", jobs[0].Key())
		assert.Equal(t, "key2", jobs[1].Key())
	})

	t.Run("should only retrieve pending jobs", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}, asyncCommand{
				Key__:   "key2",
				Topic__: "group2",
			}))
		jobs, err := store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)

		jobs, err = store.GetNextPendingJobs(context.Background())

		assert.Nil(t, err)
		assert.HasLength(t, 0, jobs, "should be empty since jobs has already been retrieved and marked has started in the process")
	})

	t.Run("should delete successfully done jobs", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}))
		jobs, err := store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 1, jobs)

		job := jobs[0]
		err = job.HasEnded(nil)

		assert.Nil(t, err)

		err = store.Write(context.Background(), &job)

		assert.Nil(t, err)

		data, err := store.GetAllJobs(context.Background(), get_jobs.Query{})

		assert.Nil(t, err)
		assert.HasLength(t, 0, data.Data)
	})

	t.Run("should delay errored jobs with the correct errcode", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}))
		jobs, err := store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 1, jobs)

		job := jobs[0]
		err = job.HasEnded(errors.New("failed"))
		assert.Nil(t, err)

		err = store.Write(context.Background(), &job)
		assert.Nil(t, err)

		data, err := store.GetAllJobs(context.Background(), get_jobs.Query{})

		assert.Nil(t, err)
		assert.HasLength(t, 1, data.Data)

		jobData := data.Data[0]
		assert.Equal(t, "failed", jobData.State.ErrorCode.Get(""))
		assert.Equal(t, embedded.JobStatusPending, embedded.JobStatus(jobData.State.Status))
	})

	t.Run("should delay jobs in the same topic", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			},
			asyncCommand{
				Key__:   "key2",
				Topic__: "group1",
			},
			asyncCommand{
				Key__:   "key3",
				Topic__: "group1",
			},
			asyncCommand{
				Key__:   "key4",
				Topic__: "group2",
			}))
		jobs, err := store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)
		assert.Equal(t, "key1", jobs[0].Key())

		job := jobs[0]
		assert.Nil(t, job.Delay())
		assert.Nil(t, store.Write(context.Background(), &job))

		jobs, err = store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 0, jobs, "should return no job since the first one has been delayed and the group2 job is running")

		data, err := store.GetAllJobs(context.Background(), get_jobs.Query{})
		assert.Nil(t, err)
		assert.HasLength(t, 4, data.Data)
		assert.Equal(t, "key1", data.Data[0].Key)
		assert.Equal(t, embedded.JobStatusPending, embedded.JobStatus(data.Data[0].State.Status))
	})

	t.Run("should reset running jobs", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}, asyncCommand{
				Key__:   "key2",
				Topic__: "group2",
			}))
		jobs, err := store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)

		err = store.ResetRetrievedJobs(context.Background())
		assert.Nil(t, err)

		jobs, err = store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)
	})

	t.Run("should be able to mark job as pending by their keys", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}, asyncCommand{
				Key__:   "key2",
				Topic__: "group2",
			}))
		jobs, err := store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)

		err = store.MarkAsPending(context.Background(), "key1")
		assert.Nil(t, err)

		jobs, err = store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 1, jobs)
		assert.Equal(t, "key1", jobs[0].Key())
	})

	t.Run("do nothing if trying to cancel an empty list of keys", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}, asyncCommand{
				Key__:   "key2",
				Topic__: "group2",
			}))

		jobs, err := store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)

		err = store.Cancel(context.Background())
		assert.Nil(t, err)

		data, err := store.GetAllJobs(context.Background(), get_jobs.Query{})
		assert.Nil(t, err)
		assert.HasLength(t, 2, data.Data)
	})

	t.Run("should delete cancelled jobs", func(t *testing.T) {
		store, _ := prepareDatabase(t)
		assert.Nil(t, store.Queue(context.Background(),
			asyncCommand{
				Key__:   "key1",
				Topic__: "group1",
			}, asyncCommand{
				Key__:   "key2",
				Topic__: "group2",
			}))

		jobs, err := store.GetNextPendingJobs(context.Background())
		assert.Nil(t, err)
		assert.HasLength(t, 2, jobs)

		err = store.Cancel(context.Background(), "key1", "key2")
		assert.Nil(t, err)

		data, err := store.GetAllJobs(context.Background(), get_jobs.Query{})
		assert.Nil(t, err)
		assert.HasLength(t, 0, data.Data)
	})

	t.Run("should update a job if an conflict occurs when writing it", func(t *testing.T) {
		t.Skip()
	})
}

func prepareDatabase(t *testing.T) (*sqlite.JobsStore, spy.Dispatcher) {
	conf := config.Default(config.WithTestDefaults())

	if err := ostools.MkdirAll(conf.DataDir()); err != nil {
		t.Fatal(err)
	}

	dispatcher := spy.NewDispatcher()

	db, err := sq.Open(conf.ConnectionString(), must.Panic(log.NewLogger()), dispatcher)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(conf.DataDir())
	})

	if err := db.Migrate(sqlite.Migrations); err != nil {
		t.Fatal(err)
	}

	return sqlite.NewJobsStore(db, dispatcher), dispatcher
}

type asyncCommand struct {
	bus.AsyncCommand

	Key__   bus.JobKey `json:"key"`
	Topic__ string     `json:"topic"`
}

func (asyncCommand) Name_() string      { return "AsyncCommand" }
func (c asyncCommand) Topic_() string   { return c.Topic__ }
func (c asyncCommand) Key_() bus.JobKey { return c.Key__ }
