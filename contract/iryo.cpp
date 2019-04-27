#include <eosio/eosio.hpp>

using namespace eosio;

class [[eosio::contract]] iryo : public contract {
  public:
      using contract::contract;

      [[eosio::action]]
      void grantaccess( name patient, name connectedUser ) {
        require_auth( patient );

        personInx persons( get_self(), patient.value );

        // check if access is already granted
        auto iterator = persons.find(connectedUser.value);
        eosio::check(iterator == persons.end(), "Access already granted");

        // add to table
        persons.emplace(patient, [&]( auto& row ) {
          row.key = connectedUser;
        });

        // check if access was granted
        iterator = persons.find(connectedUser.value);
        eosio::check(iterator != persons.end(), "Failed to grant access");
      }

      [[eosio::action]]
      void revokeaccess( name patient, name connectedUser ) {
        require_auth( patient );

        personInx persons( get_self(), patient.value );

        // check access was granted in the past
        auto iterator = persons.find(connectedUser.value);
        eosio::check(iterator != persons.end(), "Did not find a connection to revoke");

        // remove row
        persons.erase(iterator);

        // check access was revoked
        iterator = persons.find(connectedUser.value);
        eosio::check(iterator == persons.end(), "Connection not removed");
      }

      [[eosio::action]]
      void revokeaccess2( name patient, name connectedUser ) {
        require_auth( connectedUser );

        personInx persons( get_self(), patient.value );

        // check access was granted in the past
        auto iterator = persons.find(connectedUser.value);
        eosio::check(iterator != persons.end(), "Did not find a connection to revoke");

        // remove row
        persons.erase(iterator);

        // check access was revoked
        iterator = persons.find(connectedUser.value);
        eosio::check(iterator == persons.end(), "Connection not removed");
      }

  private:
    struct [[eosio::table]] person {
      name key;
      uint64_t primary_key() const { return key.value;}
    };

    typedef eosio::multi_index<"person"_n, person> personInx;
};
