package node

import "time"

// Store is the node's data access interface.
// All methods are synchronous; the SQLite implementation serialises via a mutex.
type Store interface {
	// Offers
	CreateOffer(o *Offer) error
	GetOfferByID(id int64) (*Offer, error)
	GetLatestOfferByPhone(phone string) (*Offer, error)
	GetOffersByStatus(status string) ([]*Offer, error)
	UpdateOfferStatus(id int64, status string) error
	ExpireOffers(before time.Time) (int64, error)

	// Needs
	CreateNeed(n *Need) error
	GetNeedByID(id int64) (*Need, error)
	GetLatestNeedByPhone(phone string) (*Need, error)
	GetNeedsByStatus(status string) ([]*Need, error)
	UpdateNeedStatus(id int64, status string) error
	ExpireNeeds(before time.Time) (int64, error)

	// Jobs
	CreateJob(j *Job) error
	GetJobByOfferID(offerID int64) (*Job, error)
	UpdateJobStatus(id int64, status string) error
}
