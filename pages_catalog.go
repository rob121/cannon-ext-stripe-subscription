package main

import "github.com/rob121/cannon/extension"

func (app *stripeApp) pageCatalog() (extension.PageListResponse, error) {
	pages, err := app.pageDefinitions()
	if err != nil {
		return extension.PageListResponse{}, err
	}
	return extension.PageListResponse{
		Pages: pages,
		Routes: []extension.PageRouteSpec{
			{PageID: "product", Path: "/subscribe", Name: "Subscription Product"},
			{PageID: "account", Path: "/subscribe/account", Name: "My Subscriptions"},
			{PageID: "return", Path: "/subscribe/return", Name: "Checkout Return"},
		},
	}, nil
}
