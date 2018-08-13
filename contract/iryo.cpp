#include <eosiolib/eosio.hpp>
using namespace eosio;

class iryo : public eosio::contract {
    public:
        using contract::contract;

        /// @abi action 
        void give( account_name from, account_name to ) {
            require_auth( from );
            stats statstable( _self, from );
            auto itr = statstable.find(to);
            eosio_assert(itr == statstable.end(), "Connection already exists");
            statstable.emplace( _self, [&]( auto& s ) {s.to = to;});
        }
        
        /// @abi action 
        void premove( account_name from, account_name to ) {
            require_auth( from );
            stats statstable( _self, from );
            auto itr = statstable.find(to);
            eosio_assert(itr != statstable.end(), "Connection not found");
            statstable.erase( itr );

            itr = statstable.find(to);
            eosio_assert(itr == statstable.end(), "Connection not erased properly");
            }
        /// @abi action 
        void dremove( account_name from, account_name to ) {
            require_auth( to );
            stats statstable( _self, from );
            auto itr = statstable.find(to);
            eosio_assert(itr != statstable.end(), "Connection not found");
            statstable.erase( itr );

            itr = statstable.find(to);
            eosio_assert(itr == statstable.end(), "Connection not erased properly");
            }

    private:
        //@abi table status i64
        struct status {
            account_name    to;

            uint64_t primary_key()const { return to; }
         };
        typedef eosio::multi_index<N(status), status> stats;
};

EOSIO_ABI( iryo, (give) (premove) (dremove) )
