//
// Created by Nejc
//

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
        void add(account_name patient, account_name account, uint64_t isDoctorValue, int64_t isenabledValue) {
            require_auth( patient );

            personInx persons(_self, patient);

            //
            // If person exsists, then return false
            //
            auto iterator = persons.find(account);
            eosio_assert(iterator == persons.end(), "Address for account already exists");

            persons.emplace(patient, [&](auto& person) {
                person.account_name = account;
                person.isDoctor = isDoctorValue;
				person.isEnabled = isenabledValue;
            });
        }
		
		

        //@abi action
        void update(account_name patient, account_name account, uint64_t isDoctorValue, int64_t isenabledValue) {
            require_auth(patient);

            personInx persons(_self, patient);

            auto iterator = persons.find(account);
            eosio_assert(iterator != persons.end(), "Address for account not found");

            // We add the new person in the table
            
            persons.modify(iterator, patient, [&](auto& person) {
                person.isDoctor = isDoctorValue;
				person.isEnabled = isenabledValue;
            });
        }

		//@abi action
        void grantaccess(account_name patient, account_name account) {
            require_auth( patient );
            
            personInx persons(_self, patient);

            auto iterator = persons.find(account);
            eosio_assert(iterator != persons.end(), "Address for account not found");

            //
            // We add the new person in the table
            //
            persons.modify(iterator, patient, [&](auto& person) {
				person.isEnabled = 1;
            });
        }
		
		//@abi action
        void revokeaccess(account_name patient, account_name account) {
            require_auth( patient );
            
            personInx persons(_self, patient);

            auto iterator = persons.find(account);
            eosio_assert(iterator != persons.end(), "Address for account not found");

            //
            // We add the new person in the table
            //
            persons.modify(iterator, patient, [&](auto& person) {
				person.isEnabled = 0;
            });
        }
		
		//@abi action
        void revokeaccess2(account_name patient, account_name account) {
            require_auth( account );
            
            personInx persons(_self, patient);

            auto iterator = persons.find(account);
            eosio_assert(iterator != persons.end(), "Address for account not found");
            
            //
            // We add the new person in the table
            //
            persons.modify(iterator, account, [&](auto& person) {
				person.isEnabled = 0;
            });
        }

        //@abi action
        void getperson(account_name patient, const account_name account) {
            personInx persons(_self, patient);

            auto iterator = persons.find(account);
            eosio_assert(iterator != persons.end(), "Address for account not found");

            auto currentPerson = persons.get(account);
            print("Is doctor: ", currentPerson.isDoctor, "Is enabled: ", currentPerson.isEnabled);
        }

    private:

        //@abi table person i64
        struct person {
            account_name account_name;
            uint64_t isDoctor = 0;
			uint64_t isEnabled = 0;

            uint64_t primary_key() const { return account_name; }

            EOSLIB_SERIALIZE(person, (account_name)(isDoctor)(isEnabled))
        };

        typedef multi_index<N(person), person> personInx;
    };

    EOSIO_ABI(Person, (add)(update)(grantaccess)(revokeaccess)(revokeaccess2)(getperson))
}