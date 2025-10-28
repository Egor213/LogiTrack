package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	repository_mock "github.com/Egor213/LogiTrack/internal/mocks/repository"
	"github.com/Egor213/LogiTrack/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestLogService_GetStats(t *testing.T) {
	type args struct {
		ctx     context.Context
		service string
		from    time.Time
		to      time.Time
	}

	type mockBehavior func(r *repository_mock.MockLog, args args)

	now := time.Now()
	testCases := []struct {
		name         string
		args         args
		mockBehavior mockBehavior
		want         domain.ServiceStats
		wantErr      bool
	}{
		{
			name: "success",
			args: args{
				ctx:     context.Background(),
				service: "auth",
				from:    now.Add(-time.Hour),
				to:      now,
			},
			mockBehavior: func(r *repository_mock.MockLog, args args) {
				r.EXPECT().
					GetStatsByService(args.ctx, args.service, args.from, args.to).
					Return(domain.ServiceStats{
						Service:   "auth",
						TotalLogs: 100,
						LogsByLevel: []domain.LevelStats{
							{"INFO", 80},
							{"ERROR", 20},
						},
					}, nil)
			},
			want: domain.ServiceStats{
				Service:   "auth",
				TotalLogs: 100,
				LogsByLevel: []domain.LevelStats{
					{"INFO", 80},
					{"ERROR", 20},
				},
			},
			wantErr: false,
		},
		{
			name: "repository error",
			args: args{
				ctx:     context.Background(),
				service: "billing",
				from:    now.Add(-2 * time.Hour),
				to:      now,
			},
			mockBehavior: func(r *repository_mock.MockLog, args args) {
				r.EXPECT().
					GetStatsByService(args.ctx, args.service, args.from, args.to).
					Return(domain.ServiceStats{}, errors.New("db error"))
			},
			want:    domain.ServiceStats{},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository_mock.NewMockLog(ctrl)
			tc.mockBehavior(mockRepo, tc.args)

			cnt := metrics.NewTestCounters()
			s := service.NewLogService(mockRepo, cnt)

			got, err := s.GetStats(tc.args.ctx, tc.args.service, tc.args.from, tc.args.to)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLogService_SendLog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository_mock.NewMockLog(ctrl)
	mockCounters := metrics.NewTestCounters()

	svc := service.NewLogService(mockRepo, mockCounters)

	ctx := context.Background()
	logEntry := &domain.LogEntry{
		Service:   "auth",
		Level:     "WARN",
		Message:   "something happened",
		Timestamp: time.Now(),
	}

	tcs := []struct {
		name         string
		mockBehavior func()
		wantID       int
		wantErr      bool
	}{
		{
			name: "success",
			mockBehavior: func() {
				mockRepo.EXPECT().
					SendLog(ctx, logEntry).
					Return(42, nil)
			},
			wantID:  42,
			wantErr: false,
		},
		{
			name: "repository error",
			mockBehavior: func() {
				mockRepo.EXPECT().
					SendLog(ctx, logEntry).
					Return(0, errors.New("db error"))
			},
			wantID:  0,
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockBehavior()

			gotID, err := svc.SendLog(ctx, logEntry)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantID, gotID)
		})
	}
}
