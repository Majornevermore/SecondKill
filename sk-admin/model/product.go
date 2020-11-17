package model

import (
	"SecondKill/pkg/mysql"
	"github.com/gohouse/gorose/v2"
	"log"
)

type Product struct {
	ProductId   int    `json:"product+id"`
	ProductName string `json:"product_name"`
	Total       int    `json:"total"`
	Status      int    `json:"status"`
}

type ProductModel struct {
}

func NewProductModel() *ProductModel {
	return &ProductModel{}
}

func (p *ProductModel) getTableName() string {
	return "product"
}

func (p *ProductModel) CreateProduct(product *Product) error {
	conn := mysql.DB()
	_, err := conn.Table(p.getTableName()).Data(map[string]interface{}{
		"product_name": product.ProductName,
		"total":        product.Total,
		"status":       product.Status,
	}).Insert()
	if err != nil {
		log.Printf("Error : %v", err)
		return err
	}
	return nil
}

func (p *ProductModel) GetProductList() ([]gorose.Data, error) {
	conn := mysql.DB()
	list, err := conn.Table(p.getTableName()).Get()
	if err != nil {
		log.Printf("Error : %v", err)
		return nil, err
	}
	return list, nil
}
