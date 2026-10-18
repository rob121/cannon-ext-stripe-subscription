package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/extension"
	"gorm.io/gorm"
)

type stripeSettings struct {
	Mode               string `json:"mode"`
	SecretKeyLive      string `json:"secret_key_live"`
	SecretKeyTest      string `json:"secret_key_test"`
	PublishableKeyLive string `json:"publishable_key_live"`
	PublishableKeyTest string `json:"publishable_key_test"`
	WebhookSecret      string `json:"webhook_secret"`
	SuccessPath        string `json:"success_path"`
	CancelPath         string `json:"cancel_path"`
}

type product struct {
	ID                uint      `gorm:"primaryKey;column:product_id"`
	Name              string    `gorm:"column:name;size:255;not null"`
	Description       string    `gorm:"column:description;type:text"`
	Cost              float64   `gorm:"column:cost;not null;default:0"`
	Period            string    `gorm:"column:period;size:32;not null;default:monthly"`
	StripePriceID     string    `gorm:"column:stripe_price_id;size:128"`
	StripeTestPriceID string    `gorm:"column:stripe_test_price_id;size:128"`
	TestMode          bool      `gorm:"column:test_mode;not null;default:false"`
	Status            string    `gorm:"column:status;size:16;not null;default:active;index"`
	Sort              int       `gorm:"column:sort;not null;default:0;index"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

type subscriber struct {
	ID                   uint       `gorm:"primaryKey;column:subscriber_id"`
	UserID               uint       `gorm:"column:user_id;index;not null"`
	ProductID            uint       `gorm:"column:product_id;index;not null"`
	StripeCustomerID     string     `gorm:"column:stripe_customer_id;size:128;index"`
	StripeSubscriptionID string     `gorm:"column:stripe_subscription_id;size:128;uniqueIndex"`
	Status               string     `gorm:"column:status;size:32;not null;default:incomplete;index"`
	StartAt              *time.Time `gorm:"column:start_at"`
	EndAt                *time.Time `gorm:"column:end_at"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
}

type configStore struct {
	table string
}

func (s configStore) Load(section string) (map[string]any, error) {
	db, _, err := extension.OpenDB()
	if err != nil {
		return nil, err
	}
	return extension.DBConfigurationStore(db, s.table).Load(section)
}

func (s configStore) Save(section string, data map[string]any) error {
	db, _, err := extension.OpenDB()
	if err != nil {
		return err
	}
	if data != nil {
		delete(data, "_csrf")
		delete(data, "csrf")
	}
	return extension.DBConfigurationStore(db, s.table).Save(section, data)
}

func (app *stripeApp) db() (*gorm.DB, error) {
	db, _, err := extension.OpenDB()
	return db, err
}

func (app *stripeApp) migrateTables(db *gorm.DB) error {
	app.schemaOnce.Do(func() {
		app.schemaErr = extension.MigrateConfigurationStore(db, app.settingsTable)
		if app.schemaErr != nil {
			return
		}
		if err := db.Table(app.productsTable).AutoMigrate(&product{}); err != nil {
			app.schemaErr = err
			return
		}
		app.schemaErr = db.Table(app.subscribersTable).AutoMigrate(&subscriber{})
	})
	return app.schemaErr
}

func (app *stripeApp) loadSettings() (stripeSettings, error) {
	data, err := (configStore{table: app.settingsTable}).Load("general")
	if err != nil {
		return stripeSettings{}, err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return stripeSettings{}, err
	}
	var cfg stripeSettings
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return stripeSettings{}, err
	}
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = "test"
	}
	return cfg, nil
}

func (app *stripeApp) activeProductOptions() []extension.PageFieldOption {
	db, err := app.db()
	if err != nil {
		return nil
	}
	var rows []product
	if err := db.Table(app.productsTable).Where("status = ?", "active").Order("sort asc, name asc").Find(&rows).Error; err != nil {
		return nil
	}
	out := make([]extension.PageFieldOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, extension.PageFieldOption{
			Value: strconv.FormatUint(uint64(row.ID), 10),
			Label: row.Name,
		})
	}
	return out
}

func (app *stripeApp) loadProduct(id uint) (product, error) {
	db, err := app.db()
	if err != nil {
		return product{}, err
	}
	var row product
	err = db.Table(app.productsTable).Where("product_id = ?", id).First(&row).Error
	return row, err
}

func (app *stripeApp) loadActiveProduct(id uint) (product, error) {
	db, err := app.db()
	if err != nil {
		return product{}, err
	}
	var row product
	err = db.Table(app.productsTable).Where("product_id = ? AND status = ?", id, "active").First(&row).Error
	return row, err
}

func (p product) priceID() string {
	// Prefer explicit test price when the product is flagged for test mode.
	if p.TestMode && strings.TrimSpace(p.StripeTestPriceID) != "" {
		return strings.TrimSpace(p.StripeTestPriceID)
	}
	return strings.TrimSpace(p.StripePriceID)
}

func (app *stripeApp) findSubscriberByStripeID(subID string) (subscriber, error) {
	db, err := app.db()
	if err != nil {
		return subscriber{}, err
	}
	var row subscriber
	err = db.Table(app.subscribersTable).Where("stripe_subscription_id = ?", subID).First(&row).Error
	return row, err
}

func (app *stripeApp) findActiveForUserProduct(userID, productID uint) (subscriber, bool, error) {
	db, err := app.db()
	if err != nil {
		return subscriber{}, false, err
	}
	var row subscriber
	err = db.Table(app.subscribersTable).
		Where("user_id = ? AND product_id = ? AND status IN ?", userID, productID, []string{"active", "trialing", "past_due"}).
		Order("subscriber_id desc").
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return subscriber{}, false, nil
	}
	if err != nil {
		return subscriber{}, false, err
	}
	return row, true, nil
}

func (app *stripeApp) upsertSubscriber(row subscriber) error {
	db, err := app.db()
	if err != nil {
		return err
	}
	if row.StripeSubscriptionID != "" {
		var existing subscriber
		err := db.Table(app.subscribersTable).Where("stripe_subscription_id = ?", row.StripeSubscriptionID).First(&existing).Error
		if err == nil {
			row.ID = existing.ID
			row.CreatedAt = existing.CreatedAt
			return db.Table(app.subscribersTable).Save(&row).Error
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}
	return db.Table(app.subscribersTable).Create(&row).Error
}

type userAccount struct {
	ID         uint   `gorm:"column:user_id"`
	GivenName  string `gorm:"column:given_name"`
	FamilyName string `gorm:"column:family_name"`
	Username   string `gorm:"column:username"`
	Email      string `gorm:"column:email"`
}

func (u userAccount) DisplayName() string {
	name := strings.TrimSpace(strings.TrimSpace(u.GivenName) + " " + strings.TrimSpace(u.FamilyName))
	if name != "" {
		return name
	}
	if u.Username != "" {
		return u.Username
	}
	if u.Email != "" {
		return u.Email
	}
	return "User #" + strconv.FormatUint(uint64(u.ID), 10)
}

func (u userAccount) Label() string {
	name := u.DisplayName()
	if u.Username != "" && !strings.EqualFold(name, u.Username) {
		return name + " (" + u.Username + ")"
	}
	return name
}

func (app *stripeApp) loadUserAccounts(db *gorm.DB, userIDs []uint) map[uint]userAccount {
	out := map[uint]userAccount{}
	if len(userIDs) == 0 {
		return out
	}
	seen := map[uint]struct{}{}
	ids := make([]uint, 0, len(userIDs))
	for _, id := range userIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return out
	}
	var rows []userAccount
	_ = db.Table("users").
		Select("user_id, given_name, family_name, username, email").
		Where("user_id IN ?", ids).
		Find(&rows).Error
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

func (app *stripeApp) loadUserAccount(db *gorm.DB, userID uint) (userAccount, bool) {
	if userID == 0 {
		return userAccount{}, false
	}
	accounts := app.loadUserAccounts(db, []uint{userID})
	row, ok := accounts[userID]
	return row, ok
}
