package service

import (
	"context"
	"fmt"

	"go-gophkeeper/internal/crypto"
	"go-gophkeeper/internal/domain/interfaces"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SyncService — сервис синхронизации локального кеша с сервером.
// Push отправляет локальные изменения на сервер.
// Pull загружает изменения с сервера в локальный кеш.
type SyncService struct {
	localRepo  *LocalRepository
	remoteRepo interfaces.SecretRepository
	crypto     crypto.CryptoEngine
	userID     uuid.UUID
	logger     *zap.Logger
}

// NewSyncService создаёт новый SyncService.
func NewSyncService(
	localRepo *LocalRepository,
	remoteRepo interfaces.SecretRepository,
	c crypto.CryptoEngine,
	userID uuid.UUID,
	logger *zap.Logger,
) *SyncService {
	return &SyncService{
		localRepo:  localRepo,
		remoteRepo: remoteRepo,
		crypto:     c,
		userID:     userID,
		logger:     logger,
	}
}

// Push отправляет локальные изменения на сервер.
// Возвращает количество отправленных секретов.
func (s *SyncService) Push(ctx context.Context) (int, error) {
	pushed := 0

	// 1. Отправляем новые секреты (pending_create).
	creates, err := s.localRepo.ListByStatus(ctx, SyncStatusPendingCreate)
	if err != nil {
		return pushed, fmt.Errorf("sync push: list creates: %w", err)
	}
	for _, ls := range creates {
		if err := s.remoteRepo.Create(ctx, ls.Secret); err != nil {
			s.logger.Warn("sync push: create failed",
				zap.String("secret_id", ls.ID.String()),
				zap.Error(err))
			continue
		}
		if err := s.localRepo.SetStatus(ctx, ls.ID, SyncStatusSynced); err != nil {
			s.logger.Warn("sync push: mark synced failed",
				zap.String("secret_id", ls.ID.String()),
				zap.Error(err))
		}
		pushed++
	}

	// 2. Отправляем обновления (pending_update).
	updates, err := s.localRepo.ListByStatus(ctx, SyncStatusPendingUpdate)
	if err != nil {
		return pushed, fmt.Errorf("sync push: list updates: %w", err)
	}
	for _, ls := range updates {
		if err := s.remoteRepo.Update(ctx, ls.Secret); err != nil {
			s.logger.Warn("sync push: update failed",
				zap.String("secret_id", ls.ID.String()),
				zap.Error(err))
			continue
		}
		if err := s.localRepo.SetStatus(ctx, ls.ID, SyncStatusSynced); err != nil {
			s.logger.Warn("sync push: mark synced failed",
				zap.String("secret_id", ls.ID.String()),
				zap.Error(err))
		}
		pushed++
	}

	// 3. Отправляем удаления (pending_delete).
	deletes, err := s.localRepo.ListByStatus(ctx, SyncStatusPendingDelete)
	if err != nil {
		return pushed, fmt.Errorf("sync push: list deletes: %w", err)
	}
	for _, ls := range deletes {
		if err := s.remoteRepo.Delete(ctx, ls.ID, s.userID); err != nil {
			s.logger.Warn("sync push: delete failed",
				zap.String("secret_id", ls.ID.String()),
				zap.Error(err))
			continue
		}
		if err := s.localRepo.Remove(ctx, ls.ID); err != nil {
			s.logger.Warn("sync push: remove local failed",
				zap.String("secret_id", ls.ID.String()),
				zap.Error(err))
		}
		pushed++
	}

	return pushed, nil
}

// Pull загружает изменения с сервера в локальный кеш.
// Возвращает количество полученных/обновлённых секретов.
func (s *SyncService) Pull(ctx context.Context) (int, error) {
	pulled := 0

	// 1. Получаем список мета-информации с сервера.
	remoteMetas, err := s.remoteRepo.ListByUser(ctx, s.userID)
	if err != nil {
		return pulled, fmt.Errorf("sync pull: list remote: %w", err)
	}

	// Строим map серверных ID для быстрого поиска.
	remoteIDs := make(map[uuid.UUID]struct{}, len(remoteMetas))
	for _, m := range remoteMetas {
		remoteIDs[m.ID] = struct{}{}
	}

	// 2. Для каждого секрета на сервере, которого нет локально — загружаем.
	for _, meta := range remoteMetas {
		localSecret, err := s.localRepo.GetByID(ctx, meta.ID)
		if err != nil {
			// Секрета нет локально — загружаем с сервера.
			remoteSecret, err := s.remoteRepo.GetByID(ctx, meta.ID)
			if err != nil {
				s.logger.Warn("sync pull: get remote secret failed",
					zap.String("secret_id", meta.ID.String()),
					zap.Error(err))
				continue
			}

			if err := s.localRepo.PullReplace(ctx, remoteSecret); err != nil {
				s.logger.Warn("sync pull: save local failed",
					zap.String("secret_id", meta.ID.String()),
					zap.Error(err))
				continue
			}
			pulled++
			continue
		}

		// Секрет есть локально — проверяем, нужно ли обновить.
		// Пропускаем, если локальный секрет в статусе pending_* (будет отправлен Push).
		ls, err := s.localRepo.GetLocalByID(ctx, meta.ID)
		if err != nil {
			continue
		}
		if ls.SyncStatus == SyncStatusPendingCreate ||
			ls.SyncStatus == SyncStatusPendingUpdate ||
			ls.SyncStatus == SyncStatusPendingDelete {
			continue
		}

		// Серверная версия новее — загружаем.
		if meta.CreatedAt.After(localSecret.UpdatedAt) {
			remoteSecret, err := s.remoteRepo.GetByID(ctx, meta.ID)
			if err != nil {
				s.logger.Warn("sync pull: get remote secret for update failed",
					zap.String("secret_id", meta.ID.String()),
					zap.Error(err))
				continue
			}

			if err := s.localRepo.PullReplace(ctx, remoteSecret); err != nil {
				s.logger.Warn("sync pull: replace local failed",
					zap.String("secret_id", meta.ID.String()),
					zap.Error(err))
				continue
			}
			pulled++
		}
	}

	// 3. Удаляем локально те секреты, которых нет на сервере и которые не pending.
	localMetas, err := s.localRepo.ListByUser(ctx, s.userID)
	if err != nil {
		return pulled, fmt.Errorf("sync pull: list local: %w", err)
	}

	for _, localMeta := range localMetas {
		if _, exists := remoteIDs[localMeta.ID]; !exists {
			// Проверяем статус — не удаляем pending.
			ls, err := s.localRepo.GetLocalByID(ctx, localMeta.ID)
			if err != nil {
				continue
			}
			if ls.SyncStatus != SyncStatusPendingCreate &&
				ls.SyncStatus != SyncStatusPendingUpdate &&
				ls.SyncStatus != SyncStatusPendingDelete {
				_ = s.localRepo.Remove(ctx, localMeta.ID)
				pulled++
			}
		}
	}

	return pulled, nil
}
