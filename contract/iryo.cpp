#include <eosiolib/eosio.hpp>
#include <eosiolib/print.hpp>
#include <string>

namespace ZP {
    using namespace eosio;
    using std::string;

    class Person : public contract {
        using contract::contract;

    public:
        Person(account_name self):contract(self) {}
	
		//@abi action
        void grantaccess(account_name patient, account_name account) {
            require_auth( patient );
            
            personInx persons(_self, patient);

            auto itr = persons.find(account);
            eosio_assert(itr == persons.end(), "Access already granted");

            //
            // We add the new person in the table
            //
            persons.emplace(patient, [&](auto& person) {
                person.account_name = account;
            });

            itr = persons.find(account);
            eosio_assert(itr != persons.end(), "Connection not created properly");

        }
		
		//@abi action
        void revokeaccess(account_name patient, account_name account) {
            require_auth( patient );
            
            personInx persons(_self, patient);

            auto itr = persons.find(account);
            eosio_assert(itr != persons.end(), "Address for account not found");

            //
            // We remove account from the table
            //
            persons.erase(itr);
            
            // 
            // Check if value was erased
            //
            itr = persons.find(account);
            eosio_assert(itr == persons.end(), "Connection not erased properly");

        }
		
		//@abi action
        void revokeaccess2(account_name patient, account_name account) {
            require_auth( account );
            
            personInx persons(_self, patient);

            auto itr = persons.find(account);
            eosio_assert(itr != persons.end(), "Address for account not found");

            //
            // We remove account from the table
            //
            persons.erase(itr);
            
            // 
            // Check if value was erased
            //
            itr = persons.find(account);
            eosio_assert(itr == persons.end(), "Connection not erased properly");

        }

    private:
        //@abi table person i64
        struct person {
            account_name account_name;

            uint64_t primary_key() const { return account_name; }

            EOSLIB_SERIALIZE(person, (account_name))
        };

        typedef multi_index<N(person), person> personInx;
    };

    EOSIO_ABI(Person, (grantaccess)(revokeaccess)(revokeaccess2))
}