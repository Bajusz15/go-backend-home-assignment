package service

import (
	"context"

	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
)

type RestaurantService struct {
	restaurantRepo restaurantRepository
}

func NewRestaurantService(restaurantRepo restaurantRepository) *RestaurantService {
	return &RestaurantService{restaurantRepo: restaurantRepo}
}

func (s *RestaurantService) List(ctx context.Context) ([]model.Restaurant, error) {
	return s.restaurantRepo.List(ctx)
}

func (s *RestaurantService) GetByID(ctx context.Context, id string) (*model.RestaurantWithMenu, error) {
	restaurant, err := s.restaurantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	menu, err := s.restaurantRepo.FindMenuItems(ctx, id)
	if err != nil {
		return nil, err
	}
	if menu == nil {
		menu = []model.MenuItem{}
	}

	return &model.RestaurantWithMenu{
		Restaurant: *restaurant,
		Menu:       menu,
	}, nil
}

func (s *RestaurantService) GetMenu(ctx context.Context, restaurantID string) ([]model.MenuItem, error) {
	if _, err := s.restaurantRepo.FindByID(ctx, restaurantID); err != nil {
		return nil, err
	}

	items, err := s.restaurantRepo.FindMenuItems(ctx, restaurantID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.MenuItem{}
	}
	return items, nil
}
